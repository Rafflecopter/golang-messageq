package messageq

import (
	"github.com/garyburd/redigo/redis"
  "github.com/Rafflecopter/golang-relyq/relyq"
  "github.com/yanatan16/gowaiter"
	"testing"
  "time"
  "math/rand"
  "io"
  "reflect"
)

var pool *redis.Pool

func init() {
  rand.Seed(time.Now().Unix())
	pool = redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	}, 10)
}

// -- tests --

func TestBasic(t *testing.T) {
	q, r := begin(t, config)
	defer end(t, q, r)
	done := make(chan bool)

	c, err := q.Subscribe("mychan")
	if err != nil {
		t.Error(err)
	}

	go func() {
		msg := <-c
		checkMessageEqual(t, msg, Message{"hello": "world"})
		done <- true
	}()

	r.Publish("mychan", Message{"hello": "world"})

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout waiting for done")
	}

  if err = q.Unsubscribe("mychan"); err != nil {
    t.Error(err)
  }
}

func TestClose(t *testing.T) {
  q, r := begin(t, config)

  c, err := q.Subscribe("achan")
  if err != nil {
    t.Error(err)
  }

  if err := r.Publish("achan", Message{"before":"close"}); err != nil {
    t.Error(err)
  }

  time.Sleep(10*time.Millisecond)
  end(t, q, r)

  select {
  case msg, ok := <- c:
    if !ok {
      t.Error("Got close before message on c")
    }
    checkMessageEqual(t, msg, Message{"before":"close"})
  case <- time.After(50 * time.Millisecond):
    t.Error("Timeout waiting for c message")
  }

  select {
  case _, ok := <- c:
    if ok {
      t.Error("Shouldn't be anything on c")
    }
  case <- time.After(50 * time.Millisecond):
    t.Error("Timeout waiting for c to close")
  }

  select {
  case _, ok := <- q.Errors:
    if ok {
      t.Error("Shouldn't be anything on q.Errors")
    }
  case <- time.After(50 * time.Millisecond):
    t.Error("Timeout waiting for q.Errors to close")
  }
  select {
  case _, ok := <- r.Errors:
    if ok {
      t.Error("Shouldn't be anything on r.Errors")
    }
  case <- time.After(50 * time.Millisecond):
    t.Error("Timeout waiting for r.Errors to close")
  }
}

func TestTwoWay(t *testing.T) {
  q, r := begin(t, config)
  defer end(t, q, r)
  w := waiter.New(2)

  listen := func(c chan Message, test Message) {
    for i := 0; i < 3; i++ {
      m := <- c
      checkMessageEqual(t, m, test)
    }
    w.Done <- true
  }

  if c, err := q.Subscribe("chan1"); err != nil {
    t.Error(err)
  } else {
    go listen(c, Message{"from":"q2"})
  }

  if c, err := r.Subscribe("chan2"); err != nil {
    t.Error(err)
  } else {
    go listen(c, Message{"from":"q1"})
  }

  q.Publish("chan2", Message{"from":"q1"})
  r.Publish("chan1", Message{"from":"q2"})
  r.Publish("chan1", Message{"from":"q2"})
  q.Publish("chan2", Message{"from":"q1"})
  r.Publish("chan1", Message{"from":"q2"})
  q.Publish("chan2", Message{"from":"q1"})

  if err := w.WaitTimeout(50 * time.Millisecond); err != nil {
    t.Error(err)
  }
}

func TestThreeWay(t *testing.T) {
  q, r, s := begin3(t, config)
  defer end(t, q, r, s)
  w := waiter.New(9)
  listen := func(q *MessageQueue) {
    if c, err := q.Subscribe("chan"); err != nil {
      t.Error(err)
      w.Errors <- err
    } else {
      go func () {
        froms := make(map[string]bool)
        for m := range c {
          if s, ok := m["from"].(string); ok {
            if _, ok := froms[s]; ok {
              t.Error("Got the same from twice!", froms[s], s)
            }
            froms[s] = true
          } else {
            t.Error("Couldn't cast from to string", m)
          }
          w.Done <- true
        }
      }()
    }
  }

  listen(q)
  listen(r)
  listen(s)

  if err := q.Publish("chan", Message{"from":"q1"}); err != nil {
    t.Error(err)
  }

  if err := r.Publish("chan", Message{"from":"q2"}); err != nil {
    t.Error(err)
  }

  if err := s.Publish("chan", Message{"from":"q3"}); err != nil {
    t.Error(err)
  }

  if err := w.WaitTimeout(100 * time.Millisecond); err != nil {
    t.Error(err)
  }
}

// -- helpers --

func config() *Config {
	return &Config{
		Config: &relyq.Config{
      Prefix: "go-messageq-test:" + rstr(8),
    },
	}
}

func one(t *testing.T, cfg *Config, dp string) *MessageQueue {
	mq := NewRedis(pool, cfg, dp)
  go func () {
    for err := range mq.Errors {
      t.Error(err)
    }
  }()
  return mq
}

func begin(t *testing.T, fcfg func()*Config) (*MessageQueue, *MessageQueue) {
  dp := "go-discovery-test:" + rstr(8)
  return one(t, fcfg(), dp), one(t, fcfg(), dp)
}

func begin3(t *testing.T, fcfg func()*Config) (*MessageQueue, *MessageQueue, *MessageQueue) {
  dp := "go-discovery-test:" + rstr(8)
  return one(t, fcfg(), dp), one(t, fcfg(), dp), one(t, fcfg(), dp)
}

func rstr(n int) string {
	s := make([]byte, 8)
	for i := 0; i < n; i++ {
		s[i] = byte(rand.Int()%26 + 97)
	}
	return string(s)
}

func end(t *testing.T, qs ...io.Closer) {
	for _, q := range qs {
		if err := q.Close(); err != nil {
			t.Error(err)
		}
	}
	t.Log("Active redis connections:", pool.ActiveCount())
}

func checkMessageEqual(t *testing.T, el, compare Message) {
	if id, ok := el["id"]; !ok || id == "" {
		t.Error("element has no id!", el)
	} else {
		compare["id"] = el["id"]
		if !reflect.DeepEqual(Message(el), compare) {
			t.Error("List element isn't as it should be:", el, compare)
		}
	}
}
