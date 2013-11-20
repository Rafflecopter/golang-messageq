# messageq [![Build Status][1]][2]

A simple pub/sub system that uses reliable task queues for delivery. It uses [relyq](https://github.com/Rafflecopter/golang-relyq) (simple Redis-backed reliable task queues) to establish reliable messaging. It also provides a pub/sub interface for reliable messaging by adding a discovery service backed by any database or service using a modular system.

There are a few redis clients for Go but this package uses [redigo](https://github.com/garyburd/redigo)

### Documentation

[Documentation on godoc.org](http://godoc.org/github.com/Rafflecopter/golang-messageq/messageq)

## Install

```
go get github.com/Rafflecopter/golang-messageq/messageq
```

## Creation

```go
import (
  "github.com/garyburd/redigo/redis"
  "github.com/Rafflecopter/golang-messageq/messageq"
  "time"
)

func CreateMessageQ(pool *redis.Pool) *messageq.MessageQueue {
  cfg := &messageq.Config{
    RelyQConfig: &messageq.RelyQConfig{
      Prefix: "my-relyq", // Required
      Delimiter: ":", // Defaults to :
      IdField: "id", // ID field for tasks
      UseDoneQueue: false, // Whether to keep list of "done" tasks (default false)
      KeepDoneTasks: false, // Whether to keep the backend storage of "done" tasks (default false)
    },
    // The cache decay time for listing a channels subscribers
    SubscriberListDecay: 5 * time.Minute,
  }

  // This must be the same on all nodes of a pub/sub network!
  discoveryPrefix := "my-discovery-prefix"

  return messageq.NewRedis(pool, cfg, discoveryPrefix)
}
```

## Use

```go
q := CreateMessageQ(redisPool)

if messageChannel, err := q.Subscribe("some-channel"); err != nil {
  panic(err)
} else {
  go func() {
    for message := range messageChannel {
      // Do something with your messages
    }
  }()
}

if err := q.Publish("another-channel", messageq.Message{"A":"Message"}); err != nil {
  panic(err)
}

// Eventually
if err := q.Close(); err != nil {
  panic(err)
}
```

## Tests

```
go test
```

## Discovery Backends

The messageq system can use any of the following backends, which are subclasses of the master type, so each represents a fully functional messageq system type. It is very important that all message queues on the network share the same discovery backend.

### Redis

Right now this is the only backend implemented. The Redis backend is the primary suggested one, because of its proximity to the queues. It is very important that all message queues on the network share the same `discoveryPrefix`. There's an easy creation shortcut.

```go
mq := messageq.NewRedis(pool, cfg, discoveryPrefix)
```

## License

See LICENSE file.

[1]: https://travis-ci.org/Rafflecopter/golang-messageq.png?branch=master
[2]: http://travis-ci.org/Rafflecopter/golang-messageq