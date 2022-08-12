package gosse

import (
	"fmt"
	"strings"
)

type envelope struct {
	excludeClientID string
	message         Message
}

func newEnvelope(msgFromRedis string) envelope {
	split := strings.Split(msgFromRedis, "|")
	e := envelope{excludeClientID: split[0]}
	if len(split) > 1 {
		e.message = Message(split[1])
	}
	return e
}

func newRedisMessage(message, excludeClientID string) string {
	return fmt.Sprintf("%s|%s", excludeClientID, message)
}
