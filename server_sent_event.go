package gosse

import "fmt"

type serverSentEvent struct {
	EventName        string `json:"eventName"`
	Data             string `json:"data"`
	ExcludedClientID string `json:"ecID"`
}

// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#named_events
func (s serverSentEvent) String() string {
	if s.EventName != "" && s.Data != "" {
		return fmt.Sprintf("event: %s\ndata: %s\n\n", s.EventName, s.Data)
	} else if s.Data != "" {
		return fmt.Sprintf("data: %s\n\n", s.Data)
	} else {
		return ":" // considered a comment and ignored by browser
	}
}
