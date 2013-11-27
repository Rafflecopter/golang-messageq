package main

import (
	"github.com/Rafflecopter/golang-messageq/messageq"
  "github.com/Rafflecopter/golang-messageq/example"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
  "os"
  "os/signal"
  "io"
)

func DoSubscriber(mq *messageq.MessageQueue) {
	log.Println("Subscribing to the tick channel")

  var exmsg *example.ExampleMessage
	c, err := mq.Subscribe("tick", exmsg)
	if err != nil {
		log.Panicln("Error subscribing", err)
	}

	defer func() {
    // Notice how this will run when c closes
    // which happens when we close the messageq
    // Which happens when a SIGTERM is sent to the process
		if err := mq.Unsubscribe("tick"); err != nil {
			log.Println("Error unsubscribing from tick", err)
		} else {
      log.Println("Unsubscribing from tick channel successful")
    }
	}()

	for msg := range c {
    exmsg := msg.(*example.ExampleMessage)
    exmsg.Received = time.Now().UnixNano()

    log.Println("Received message:", exmsg)
	}
}

func main() {
	pool := CreatePool()
	mq := CreateMessageQ(pool)
  CatchSignal(mq)
	DoSubscriber(mq)
}

func CreateMessageQ(pool *redis.Pool) *messageq.MessageQueue {
	cfg := &messageq.Config{
		RelyQConfig: &messageq.RelyQConfig{
			Prefix:        "subscriber-messageq", // Required
		},
	}

	// This must be the same on all nodes of a pub/sub network!
	discoveryPrefix := "my-discovery-prefix"

	return messageq.NewRedis(pool, cfg, discoveryPrefix)
}

func CreatePool() *redis.Pool {
	return redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	}, 2)
}

func CatchSignal(mq io.Closer) {
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  go func(){
    <- c
    if err := mq.Close(); err != nil {
      log.Println("Close Error", err)
    } else {
      log.Println("Close successful")
    }
  }()
}