# Go Distributed Server Sent Events

A library for server sent events in distributed environments, such as dockerized apps running multiple instances.

https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events

Clients can be connected to any app instance and they will receive events sent from any app instance using the same stream ID.

Redis is used as a message broker.

## Usage

A Broker must be instantiated and a redis client must be passed in. (redis client from github.com/go-redis/redis)

Use the included http handler.

```
broker := NewBroker(redisClient)

http.HandleFunc("/events", broker.HttpHandler())

http.ListenAndServe(":9000", nil)
```

Clients must supply a query param named "stream_id"

```
http://localhost:9000/events?stream_id=some_stream_id
```

Messages can be broadcast to all connected clients for a specified stream ID.

```
broker.Broadcast("some_stream_id", "event_name", "my message")
```

Optionally, a single client can be excluded from the broadcast.

```
broker.Broadcast("some_stream_id", "event_name" "my message", "client_id_to_exclude")
```

Optionally a client ID can be set in the request context in a middleware function that runs before the http handler. The provided context key must be used when setting the client ID.

```
newContext := context.WithValue(r.Context(), gosse.ClientIDContextKey, "the_client_id")
r = r.WithContext(newContext)
```

If a client ID is not set, one will be generated automatically.

If you don't care about excluding a client from the broadcast, there is no need to set the client ID.

