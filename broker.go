package gosse

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"sync"
	"time"
)

type Broker struct {
	streams          map[string]eventStream // key is stream ID
	streamM          sync.RWMutex
	redisClient      *redis.Client
	streamFinishedCH chan string // stream ID
}

func NewBroker(redisClient *redis.Client) *Broker {

	b := Broker{
		streams:          make(map[string]eventStream),
		redisClient:      redisClient,
		streamFinishedCH: make(chan string, 1),
	}

	go b.run()

	return &b
}

func (b *Broker) createStreamIfNotExists(streamID string) error {
	b.streamM.RLock()
	_, ok := b.streams[streamID]
	b.streamM.RUnlock()

	if ok {
		return nil
	}

	stream := newEventStream(streamID, b.streamFinishedCH, b.redisClient)

	b.streamM.Lock()
	defer b.streamM.Unlock()
	b.streams[stream.id] = stream

	return nil
}

func (b *Broker) removeStream(streamID string) error {
	b.streamM.Lock()
	defer b.streamM.Unlock()

	stream := b.streams[streamID]
	stream.Done() // exit goroutine
	delete(b.streams, streamID)

	return nil
}

// We don't need to worry about whether a stream with the passed ID exists. We know it does.
// Because this method is only called internally by an http handler created in our package.
// And that handler only has streamIDs that exist.
func (b *Broker) subscribe(streamID, clientID string) (sseClient, error) {

	err := b.createStreamIfNotExists(streamID)
	if err != nil {
		return sseClient{}, err
	}

	b.streamM.RLock()
	stream := b.streams[streamID]
	b.streamM.RUnlock()

	c := newClient(clientID, streamID, stream.clientUnsubscribedCH)

	stream.Subscribe(c)

	return c, nil
}

// Broadcast Will broadcast a message to all connected clients that are subscribed to
// the specified eventStreamID. If you want to exclude a client, so that a client does not
// receive a message, you may pass it's ID for the arg excludeClientID. Even though excludeClientID
// is a variadic argument, only the first string passed will be used.
func (b *Broker) Broadcast(eventStreamID, eventName, eventData string, excludeClientID ...string) error {
	var excludedClientID string
	if len(excludeClientID) > 0 {
		excludedClientID = excludeClientID[0]
	}

	ev := serverSentEvent{
		EventName:        eventName,
		Data:             eventData,
		ExcludedClientID: excludedClientID,
	}

	evJson, err := json.Marshal(&ev)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = b.redisClient.Publish(ctx, redisChannelName(eventStreamID), evJson).Result()
	return err
}

// HttpHandler Returns a new http handler that will send events
// to clients for specified event stream ID. stream ID is the query param "stream_id".
func (b *Broker) HttpHandler() http.HandlerFunc {
	return newSSEHttpHandler(b)
}

func (b *Broker) run() {

	for {
		select {
		case streamID := <-b.streamFinishedCH:
			err := b.removeStream(streamID)
			if err != nil {
				log.Println(err)
			}
		}

	}
}
