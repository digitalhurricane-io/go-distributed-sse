package gosse

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

// ContextKey Golang docs state that a custom type should be used when setting values in the context to
// avoid collisions with other packages
type contextKey string

// ClientIDContextKey The client ID should be stored in request context using this key.
// A middleware function should set this key before the http handler runs.
var ClientIDContextKey = contextKey("clientID")

func newSSEHttpHandler(broker *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		streamID := mux.Vars(r)["stream_id"]

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		clientID, ok := r.Context().Value(ClientIDContextKey).(string)
		if !ok {
			log.Println("clientID was not set")
			w.WriteHeader(500)
			return
		}

		client, err := broker.subscribe(streamID, clientID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

	loop:
		for {
			select {
			case msg, open := <-client.messages:
				if !open {
					break loop
				}
				_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
				if err != nil {
					log.Println(err)
				}

				flusher.Flush()

			case <-r.Context().Done():
				client.Unsubscribe()
				break
			}
		}
	}
}
