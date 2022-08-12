package gosse

import (
	"fmt"
)

const redisNameSpace = "sse|"

// namespace streamID in redis
func redisChannelName(streamID string) string {
	return fmt.Sprintf("%s%s", redisNameSpace, streamID)
}
