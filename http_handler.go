package gosse

import (
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

// ContextKey Golang docs state that a custom type should be used when setting values in the context to
// avoid collisions with other packages
type contextKey string

// ClientIDContextKey The client ID should be stored in request context using this key.
// A middleware function should set this key before the http handler runs.
var ClientIDContextKey = contextKey("clientID")

func newSSEHttpHandler(broker *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		streamID := r.URL.Query().Get("stream_id")
		if streamID == "" {
			log.Println("stream_id query param must be defined")
			w.WriteHeader(500)
			return
		}

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
			id, err := uuid.NewUUID()
			if err != nil {
				log.Println("failed to generate uuid: ", err)
				w.WriteHeader(500)
				return
			}

			clientID = id.String()
		}

		client, err := broker.subscribe(streamID, clientID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(http.StatusOK)
		flusher.Flush()

	loop:
		for {
			select {
			case ev, open := <-client.events:
				if !open {
					break loop
				}
				_, err = w.Write([]byte(ev.String()))
				if err != nil {
					log.Println(err)
				}

				flusher.Flush()

			case <-time.After(60 * time.Second):
				_, err = w.Write([]byte(":")) // send comment as keepalive
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
