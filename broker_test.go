package gosse

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"testing"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr: "127.0.0.1:44293",
})

func TestBrokerCreate(t *testing.T) {
	broker := NewBroker(redisClient)

	const streamA = "stream-a"
	const streamB = "stream-b"
	const streamC = "stream-c"
	streamNames := []string{streamA, streamB, streamC}

	for _, name := range streamNames {
		err := registerHttpHandler(broker, name)
		if err != nil {
			t.Fatal(err)
		}
	}

	go runHttpServer()

}

func doClientRequest(path string) {
	http.Get(path)
}

func runHttpServer() {
	err := http.ListenAndServe(":48923", nil)
	if err != nil {
		log.Println(err)
	}
}

func registerHttpHandler(b *Broker, streamName string) error {
	h, err := b.NewSSEHttpHandler(streamName)
	if err != nil {
		return err
	}

	http.HandleFunc(fmt.Sprintf("/%s", streamName), h)

	return nil
}
