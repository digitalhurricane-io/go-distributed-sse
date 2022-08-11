package server_sent_event

type Subscription struct {
	streamID      string
	clientID      string
	messageCH     chan Message
	unsubscribeCH chan Subscription
}

func newSubscription(streamID, clientID string, unsubscribeCH chan Subscription) Subscription {
	return Subscription{
		streamID:      streamID,
		clientID:      clientID,
		messageCH:     make(chan Message, 1),
		unsubscribeCH: unsubscribeCH,
	}
}

func (s Subscription) Channel() chan Message {
	return s.messageCH
}

func (s Subscription) Unsubscribe() {
	s.unsubscribeCH <- s
}
