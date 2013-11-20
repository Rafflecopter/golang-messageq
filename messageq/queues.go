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

func (q *queue) Messages(errs chan error) chan Message {
  if q.c == nil {
    l := q.q.Listen()
    // No failing in messageq
    go forwardErrors(l.Errors, errs)
    q.c = wrapChan(l)
  }
  return q.c
}

func wrapChan(l *relyq.Listener) chan Message {
	mc := make(chan Message)
	go func() {
		for t := range l.Tasks {
			mc <- Message(t)
      l.Finish <- t
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