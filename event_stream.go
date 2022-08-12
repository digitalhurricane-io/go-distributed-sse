package gosse

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type eventStream struct {
	id                   string
	clients              map[string]sseClient
	subscribeCH          chan sseClient
	clientUnsubscribedCH chan *sseClient

	// eventStream should send it's ID through this channel when there are no more connected clients
	streamFinishedCH chan string
	pubSub           *redis.PubSub
	done             chan struct{}
}

// todo: stream needs to track how many connected clients it has. if it has no more clients,
// it needs to signal to broker that it is no longer needed and cleanup. Each stream should
// subscribe to a redis channel.

func newEventStream(id string, streamFinishedCH chan string, redisClient *redis.Client) eventStream {

	pubSub := redisClient.Subscribe(context.TODO(), redisChannelName(id))

	s := eventStream{
		id:                   id,
		clients:              make(map[string]sseClient),
		subscribeCH:          make(chan sseClient, 1),
		clientUnsubscribedCH: make(chan *sseClient, 1),
		streamFinishedCH:     streamFinishedCH,
		pubSub:               pubSub,
		done:                 make(chan struct{}, 1),
	}

	go s.run()

	return s
}

func (s *eventStream) ID() string {
	return s.id
}

func (s *eventStream) Subscribe(c sseClient) {
	s.subscribeCH <- c
}

func (s *eventStream) Done() {
	s.done <- struct{}{}
}

func (s *eventStream) run() {
	for {
		select {
		case client := <-s.subscribeCH:
			s.clients[client.id] = client

		case client := <-s.clientUnsubscribedCH:
			close(client.messages)
			delete(s.clients, client.id)

			// if we have no more connected clients, signal to broker that it can clean up this stream
			if len(s.clients) == 0 {
				s.streamFinishedCH <- s.id
			}

		case msg := <-s.pubSub.Channel():
			e := newEnvelope(msg.Payload)
			for clientID, c := range s.clients {
				if e.excludeClientID != clientID {
					c.SendMessage(e.message)
				}
			}

		case <-s.done:
			for _, c := range s.clients {
				close(c.messages) // frees up any http handler goroutines that were listening to this event stream
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := s.pubSub.Unsubscribe(ctx, redisChannelName(s.id))
			if err != nil {
				log.Println(err)
			}

			// since we're breaking the loop, we don't have to worry that any message will be sent on
			// a closed client channel
			break
		}
	}
}
