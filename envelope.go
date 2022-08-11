package server_sent_event

import "strings"

type envelope struct {
	senderID string
	message  Message
}

func newEnvelope(msgFromRedis string) envelope {
	split := strings.Split(msgFromRedis, "|")
	e := envelope{senderID: split[0]}
	if len(split) > 1 {
		e.message = Message(split[1])
	}
	return e
}
