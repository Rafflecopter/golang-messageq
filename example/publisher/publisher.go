package main

import (
	"github.com/Rafflecopter/golang-messageq/example"
	"github.com/Rafflecopter/golang-messageq/messageq"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

func DoPublisher(mq *messageq.MessageQueue) {
	dur := 1 * time.Second
	log.Println("Publishing a tick message every", dur)

	for _ = range time.Tick(dur) {
		if err := mq.Publish("tick", &example.ExampleMessage{Name: "tick", Sent: time.Now().UnixNano()}); err != nil {
			log.Println("Error publishing tick:", err)
		} else {
			log.Println("Published tick successfully at", time.Now())
		}
	}
}

func main() {
	pool := CreatePool()
	mq := CreateMessageQ(pool)
	DoPublisher(mq)
}

func CreateMessageQ(pool *redis.Pool) *messageq.MessageQueue {
	cfg := &messageq.Config{
		RelyQConfig: &messageq.RelyQConfig{
			Prefix: "subscriber-messageq", // Required
		},
		SubscriberListDecay: time.Second, // This is low because of the example app
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
