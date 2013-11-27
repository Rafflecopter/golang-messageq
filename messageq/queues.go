package messageq

import (
	"github.com/Rafflecopter/golang-relyq/relyq"
)

type queue struct {
	q *relyq.Queue
	c chan Message
}

func (mq *MessageQueue) newQueue(endpoint string) *queue {
	cfg := new(relyq.Config)
	*cfg = relyq.Config(*mq.cfg.RelyQConfig) // Copy!
	cfg.Prefix = endpoint
	q := relyq.NewRedisJson(mq.pool, cfg)

	return &queue{
		q: q,
	}
}

func (q *queue) Close() error {
	return q.q.Close()
}

func (q *queue) Messages(errs chan error, example Message) chan Message {
	if q.c == nil {
		l := q.q.Listen(example)
		// No failing in messageq
		go forwardErrors(l.Errors, errs)
		q.c = wrapChan(l)
	}
	return q.c
}

func wrapChan(l *relyq.Listener) chan Message {
	mc := make(chan Message)
	go func() {
		for msg := range l.Tasks {
			mc <- msg
			l.Finish <- msg
		}
		close(l.Finish)
		close(l.Fail)
		close(mc)
	}()
	return mc
}

func forwardErrors(in, out chan error) {
	for err := range in {
		out <- err
	}
}
