package gosse

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"time"
)

const redisNameSpace = "sse|"

// namespace streamID in redis
func redisChannelName(streamID string) string {
	return fmt.Sprintf("%s%s", redisNameSpace, streamID)
}

func streamIDFromRedisChannelName(nameSpacedID string) string {
	return nameSpacedID[len(redisNameSpace):]
}

type Message string

type Broker struct {
	streams       map[string]eventStream // key is stream ID
	streamM       sync.RWMutex
	redisClient   *redis.Client
	pubSub        *redis.PubSub
	unsubscribeCH chan Subscription
}

func NewBroker(redisClient *redis.Client) *Broker {

	pubSub := redisClient.Subscribe(context.Background())

	b := Broker{
		streams:       make(map[string]eventStream),
		redisClient:   redisClient,
		pubSub:        pubSub,
		unsubscribeCH: make(chan Subscription),
	}

	go b.run()

	return &b
}

func (b *Broker) createStream(streamID string) error {
	b.streamM.RLock()
	_, ok := b.streams[streamID]
	b.streamM.RUnlock()

	if ok {
		return nil
	}

	nsStreamID := redisChannelName(streamID)

	err := b.pubSub.Subscribe(context.Background(), nsStreamID)
	if err != nil {
		return err
	}

	stream := newEventStream(streamID)

	b.streamM.Lock()
	defer b.streamM.Unlock()
	b.streams[stream.id] = stream

	return nil
}

func (b *Broker) removeStream(streamID string) error {
	nsStreamID := redisChannelName(streamID)
	err := b.pubSub.Unsubscribe(context.TODO(), nsStreamID)
	if err != nil {
		return err
	}

	b.streamM.RLock()
	defer b.streamM.RUnlock()

	stream := b.streams[streamID]
	stream.Done() // exit goroutine
	delete(b.streams, streamID)

	return nil
}

// We don't need to worry about whether a stream with the passed ID exists. We know it does.
// Because this method is only called internally by an http handler created in our package.
// And that handler only has streamIDs that exist.
func (b *Broker) subscribe(streamID, clientID string) Subscription {
	sub := newSubscription(streamID, clientID, b.unsubscribeCH)

	b.streamM.RLock()
	stream := b.streams[sub.streamID]
	b.streamM.RUnlock()

	stream.Subscribe(sub)

	return sub
}

// Broadcast Will broadcast a message to all connected clients that are subscribed to
// the specified eventStreamID. If you want to exclude a client, so that a client does not
// receive a message, you may pass it's ID for the arg excludeClientID. Even though excludeClientID
// is a variadic argument, only the first string passed will be used.
func (b *Broker) Broadcast(eventStreamID, message string, excludeClientID ...string) error {
	var excludedClientID string
	if len(excludeClientID) > 0 {
		excludedClientID = excludeClientID[0]
	}

	b.streamM.RLock()
	_, streamExists := b.streams[eventStreamID]
	b.streamM.RUnlock()

	if !streamExists {
		return errors.Errorf("no stream with id: %s", eventStreamID)
	}
	msg := newRedisMessage(message, excludedClientID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := b.redisClient.Publish(ctx, redisChannelName(eventStreamID), msg).Result()
	return err
}

// NewSSEHttpHandler Returns a new http handler that will send events
// to clients for specified event stream ID. If an eventStream with the
// specified streamID does not exist, one will be created.
func (b *Broker) NewSSEHttpHandler(streamID string) (http.HandlerFunc, error) {
	err := b.createStream(streamID)
	if err != nil {
		return nil, err
	}
	return newSSEHttpHandler(b, streamID), nil
}

func (b *Broker) redisMsgReceived(msg *redis.Message) {
	// not using defer in case sending on channel s.broadcastRecvCH blocks.
	// don't want to hold mutex in that case.
	b.streamM.RLock()
	s, ok := b.streams[streamIDFromRedisChannelName(msg.Channel)]
	b.streamM.RUnlock()

	if !ok {
		return
	}

	s.broadcastRecvCH <- newEnvelope(msg.Payload)
}

func (b *Broker) unsubscribe(sub Subscription) {
	b.streamM.RLock()
	stream := b.streams[sub.streamID]
	b.streamM.RUnlock()

	stream.Unsubscribe(sub)
}

func (b *Broker) run() {

	for {
		select {

		// listen for messages broadcast on redis pubsub
		case msg := <-b.pubSub.Channel():
			b.redisMsgReceived(msg)

		// unsubscribe clients
		case sub := <-b.unsubscribeCH:
			b.unsubscribe(sub)
		}

	}
}
