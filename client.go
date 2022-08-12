package gosse

type sseClient struct {
	id            string
	streamID      string
	messages      chan string
	unsubscribeCh chan *sseClient
}

func newClient(id, streamID string, unsubscribeCh chan *sseClient) sseClient {
	c := sseClient{
		id:            id,
		streamID:      streamID,
		messages:      make(chan string, 1),
		unsubscribeCh: unsubscribeCh,
	}

	return c
}

func (c *sseClient) SendMessage(message string) {
	c.messages <- message
}

// Unsubscribe Signals to eventStream that this client has disconnected and can be cleaned up
func (c *sseClient) Unsubscribe() {
	c.unsubscribeCh <- c
}
