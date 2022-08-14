package gosse

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

/**
Tests should be run by running the script test.sh which will run a redis docker container.
*/

const httpPort = "48923"

var redisClientA = redis.NewClient(&redis.Options{
	Addr: "127.0.0.1:44293",
})

var redisClientB = redis.NewClient(&redis.Options{
	Addr: "127.0.0.1:44293",
})

var redisClientC = redis.NewClient(&redis.Options{
	Addr: "127.0.0.1:44293",
})

func TestBroadcast(t *testing.T) {
	a := NewBroker(redisClientA)
	b := NewBroker(redisClientB)
	c := NewBroker(redisClientC)

	server := &http.Server{Addr: ":" + httpPort}

	http.HandleFunc("/events", a.HttpHandler())

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	receivedEvent := make(chan string, 4)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/events?stream_id=aaa", httpPort))
	if err != nil {
		t.Fatal(err)
	}

	go func() {

		scanner := bufio.NewScanner(resp.Body)
		scanner.Split(doubleNewLineSplitFunc)

		for {
			hasData := scanner.Scan()
			if !hasData {
				break
			}

			receivedEvent <- scanner.Text()
		}

	}()

	err = a.Broadcast("aaa", "my_event_1", "Yay. It works! 1")
	if err != nil {
		t.Fatal(err)
	}

	err = b.Broadcast("aaa", "my_event_1", "Yay. It works! 1")
	if err != nil {
		t.Fatal(err)
	}

	err = c.Broadcast("aaa", "my_event_2", "Yay. It works! 2")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-receivedEvent:
	case <-time.After(time.Millisecond * 50):
		t.Fatal("took too long to receive event first event")
	}

	select {
	case <-receivedEvent:
	case <-time.After(time.Millisecond * 50):
		t.Fatal("took too long to receive second event")
	}

	select {
	case <-receivedEvent:
	case <-time.After(time.Millisecond * 50):
		t.Fatal("took too long to receive third event")
	}

	err = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = server.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

}

// https://stackoverflow.com/a/33069759/6716264
func doubleNewLineSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {

	// Return nothing if at end of file and no data passed
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Find the index of the input of a newline followed by a
	// pound sign.
	if i := strings.Index(string(data), "\n\n"); i >= 0 {
		return i + 1, data[0:i], nil
	}

	// If at end of file with data return the data
	if atEOF {
		return len(data), data, nil
	}

	return
}
