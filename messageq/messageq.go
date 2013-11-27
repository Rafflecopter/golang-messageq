// Package messageq provides a reliable pub/sub bus for long term listeners job queuing
package messageq

import (
	"github.com/Rafflecopter/golang-relyq/relyq"
	"github.com/garyburd/redigo/redis"
	"github.com/yanatan16/gowaiter"
	"io"
	"time"
)

// -- Types --

// A message queue object.
type MessageQueue struct {
  Errors chan error
	pool        *redis.Pool
	cfg         *Config
	prefix      string
	queues      map[string]*queue
	discovery   Discovery
	subscribers *subscribersMemo
}

// Interface for discovery backend's to implement
type Discovery interface {
	// Register an endpoint as listening on a channel
	Register(channel, endpoint string) error
	// Unregister an endpoint as listening on a channel
	Unregister(channel, endpiont string) error
	// List subscribers to a channel
	Subscribers(channel string) ([]string, error)

	io.Closer
}

// A message in MessageQueue must have the Id() function required by tasks in RelyQueue
type Message interface {
	relyq.Ider
}

type RelyQConfig relyq.Config

type Config struct {
	*RelyQConfig

	// Subscribers cache decay duration
	SubscriberListDecay time.Duration
}

// -- Functions and Methods --

func New(pool *redis.Pool, disco Discovery, cfg *Config) *MessageQueue {
	cfg.Defaults()
	mq := &MessageQueue{
    Errors: make(chan error),
		pool:        pool,
		cfg:         cfg,
		queues:      make(map[string]*queue),
		discovery:   disco,
		subscribers: memoizeSubscriberser(disco, cfg.SubscriberListDecay),
	}

	return mq
}

// Subscribe on a channel. Returns the channel of messages
func (mq *MessageQueue) Subscribe(channel string, example Message) (chan Message, error) {
	name, q := mq.endpoint(channel)
	err := mq.discovery.Register(channel, name)

	return q.Messages(mq.Errors, example), err
}

// Publish a message on a channel
func (mq *MessageQueue) Publish(channel string, message Message) error {
	list, err := mq.subscribers.Subscribers(channel)
	if err != nil {
		return err
	}

	for _, endp := range list {
		if err2 := mq.send(endp, message); err != nil {
      err = err2
    }
	}


	return err
}

// Unsubscribe from a channel
func (mq *MessageQueue) Unsubscribe(channel string) error {
	name := mq.getEndpoint(channel)
	w := waiter.New(2)

	if q, ok := mq.queues[name]; ok {
		delete(mq.queues, name)

		w.Close(q)
	} else {
		w.Bypass()
	}

	go func() {
		w.Wrap(mq.discovery.Unregister(channel, name))
	}()

	return w.Wait()
}

func (mq *MessageQueue) Close() error {
	w := waiter.New(len(mq.queues) + 1)
	w.Close(mq.discovery)
	for _, q := range mq.queues {
		w.Close(q)
	}
	err := w.Wait()
  close(mq.Errors)
  return err
}

// -- helpers --

func (mq *MessageQueue) endpoint(channel string) (string, *queue) {
	endpoint := mq.getEndpoint(channel)
	return endpoint, mq.getQueue(endpoint)
}

func (mq *MessageQueue) getEndpoint(channel string) string {
	return mq.cfg.Prefix + mq.cfg.Delimiter + channel
}

func (mq *MessageQueue) getQueue(endpoint string) *queue {
	q, ok := mq.queues[endpoint]
	if !ok {
		q = mq.newQueue(endpoint)
		mq.queues[endpoint] = q
	}
	return q
}

func (mq *MessageQueue) send(endpoint string, message Message) error {
	q := mq.getQueue(endpoint)
	return q.q.Push(message)
}

func (rqc *RelyQConfig) Defaults() {
	cfg := new(relyq.Config)
	*cfg = relyq.Config(*rqc)
	cfg.Defaults()
	*rqc = RelyQConfig(*cfg)
}

func (cfg *Config) Defaults() {
	cfg.RelyQConfig.Defaults()

	if cfg.SubscriberListDecay == 0 {
		cfg.SubscriberListDecay = 5 * time.Minute
	}
}