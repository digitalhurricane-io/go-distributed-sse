package server_sent_event

type eventStream struct {
	id              string
	clients         map[string]chan Message
	subscribeCH     chan Subscription
	unsubscribeCH   chan Subscription
	broadcastRecvCH chan envelope
	done            chan struct{}
}

func newEventStream(id string) eventStream {

	s := eventStream{
		id:              id,
		clients:         make(map[string]chan Message),
		subscribeCH:     make(chan Subscription),
		unsubscribeCH:   make(chan Subscription),
		broadcastRecvCH: make(chan envelope, 1),
		done:            make(chan struct{}, 1),
	}

	go s.run()

	return s
}

func (s *eventStream) ID() string {
	return s.id
}

func (s *eventStream) Subscribe(sub Subscription) {
	s.subscribeCH <- sub
}

func (s *eventStream) Unsubscribe(sub Subscription) {
	s.unsubscribeCH <- sub
}

func (s *eventStream) Done() {
	s.done <- struct{}{}
}

func (s *eventStream) run() {
	for {
		select {
		case sub := <-s.subscribeCH:
			s.clients[sub.clientID] = sub.messageCH

		case sub := <-s.unsubscribeCH:
			close(sub.messageCH)
			delete(s.clients, sub.clientID)

		case e := <-s.broadcastRecvCH:
			for clientID, c := range s.clients {
				if e.senderID != clientID {
					c <- e.message
				}
			}

		case <-s.done:
			for _, c := range s.clients {
				close(c) // frees up any http handler goroutines that were listening to this event stream
			}
			break
		}
	}
}
