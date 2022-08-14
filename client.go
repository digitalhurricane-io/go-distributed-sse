package gosse

type sseClient struct {
	id            string
	streamID      string
	events        chan serverSentEvent
	unsubscribeCh chan *sseClient
}

func newClient(id, streamID string, unsubscribeCh chan *sseClient) sseClient {
	c := sseClient{
		id:            id,
		streamID:      streamID,
		events:        make(chan serverSentEvent, 1),
		unsubscribeCh: unsubscribeCh,
	}

	return c
}

func (c *sseClient) SendEvent(ev serverSentEvent) {
	c.events <- ev
}

// Unsubscribe Signals to eventStream that this client has disconnected and can be cleaned up
func (c *sseClient) Unsubscribe() {
	c.unsubscribeCh <- c
}
