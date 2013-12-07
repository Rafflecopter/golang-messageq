package messageq

import (
	"github.com/garyburd/redigo/redis"
	"github.com/yanatan16/gowaiter"
	"io"
	"math/rand"
	"reflect"
	"testing"
	"time"
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

	c, err := q.Subscribe("mychan", ArbitraryMessage{})
	if err != nil {
		t.Error(err)
	}

	go func() {
		msg := <-c
		checkMessageEqual(t, msg, ArbitraryMessage{"hello": "world"})
		done <- true
	}()

	r.Publish("mychan", ArbitraryMessage{"hello": "world"})

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

	c, err := q.Subscribe("achan", ArbitraryMessage{})
	if err != nil {
		t.Error(err)
	}

	if err := r.Publish("achan", ArbitraryMessage{"before": "close"}); err != nil {
		t.Error(err)
	}

	time.Sleep(10 * time.Millisecond)
	end(t, q, r)

	select {
	case msg, ok := <-c:
		if !ok {
			t.Error("Got close before message on c")
		}
		checkMessageEqual(t, msg, ArbitraryMessage{"before": "close"})
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout waiting for c message")
	}

	select {
	case _, ok := <-c:
		if ok {
			t.Error("Shouldn't be anything on c")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout waiting for c to close")
	}

	select {
	case _, ok := <-q.Errors:
		if ok {
			t.Error("Shouldn't be anything on q.Errors")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout waiting for q.Errors to close")
	}
	select {
	case _, ok := <-r.Errors:
		if ok {
			t.Error("Shouldn't be anything on r.Errors")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout waiting for r.Errors to close")
	}
}

func TestTwoWay(t *testing.T) {
	q, r := begin(t, config)
	defer end(t, q, r)
	w := waiter.New(2)

	listen := func(c chan Message, test Message) {
		for i := 0; i < 3; i++ {
			m := <-c
			checkMessageEqual(t, m, test)
		}
		w.Done <- true
	}

	if c, err := q.Subscribe("chan1", &StructMessage{}); err != nil {
		t.Error(err)
	} else {
		go listen(c, &StructMessage{X: "q2"})
	}

	if c, err := r.Subscribe("chan2", &StructMessage{}); err != nil {
		t.Error(err)
	} else {
		go listen(c, &StructMessage{X: "q1"})
	}

	q.Publish("chan2", &StructMessage{X: "q1"})
	r.Publish("chan1", &StructMessage{X: "q2"})
	r.Publish("chan1", &StructMessage{X: "q2"})
	q.Publish("chan2", &StructMessage{X: "q1"})
	r.Publish("chan1", &StructMessage{X: "q2"})
	q.Publish("chan2", &StructMessage{X: "q1"})

	if err := w.WaitTimeout(50 * time.Millisecond); err != nil {
		t.Error(err)
	}
}

func TestThreeWay(t *testing.T) {
	q, r, s := begin3(t, config)
	defer end(t, q, r, s)
	w := waiter.New(9)
	listen := func(q *MessageQueue) {
		if c, err := q.Subscribe("chan", ArbitraryMessage{}); err != nil {
			t.Error(err)
			w.Errors <- err
		} else {
			go func() {
				froms := make(map[string]bool)
				for msg := range c {
					if m, ok := msg.(ArbitraryMessage); !ok {
						t.Error("msg is not an ArbitraryMessage?")
					} else {
						if s, ok := m["from"].(string); ok {
							if _, ok := froms[s]; ok {
								t.Error("Got the same from twice!", froms[s], s)
							}
							froms[s] = true
						} else {
							t.Error("Couldn't cast from to string", m)
						}
					}
					w.Done <- true
				}
			}()
		}
	}

	listen(q)
	listen(r)
	listen(s)

	if err := q.Publish("chan", ArbitraryMessage{"from": "q1"}); err != nil {
		t.Error(err)
	}

	if err := r.Publish("chan", ArbitraryMessage{"from": "q2"}); err != nil {
		t.Error(err)
	}

	if err := s.Publish("chan", ArbitraryMessage{"from": "q3"}); err != nil {
		t.Error(err)
	}

	if err := w.WaitTimeout(100 * time.Millisecond); err != nil {
		t.Error(err)
	}
}

// -- helpers --

type StructMessage struct {
	StructuredMessage
	X string
}

func config() *Config {
	return &Config{
		RelyQConfig: &RelyQConfig{
			Prefix: "go-messageq-test:" + rstr(8),
		},
	}
}

func one(t *testing.T, cfg *Config, dp string) *MessageQueue {
	mq := NewRedis(pool, cfg, dp)
	go func() {
		for err := range mq.Errors {
			t.Error(err)
		}
	}()
	return mq
}

func begin(t *testing.T, fcfg func() *Config) (*MessageQueue, *MessageQueue) {
	dp := "go-discovery-test:" + rstr(8)
	return one(t, fcfg(), dp), one(t, fcfg(), dp)
}

func begin3(t *testing.T, fcfg func() *Config) (*MessageQueue, *MessageQueue, *MessageQueue) {
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
	if arb, ok := el.(ArbitraryMessage); ok {
		if id, ok := arb["id"]; !ok || id == "" {
			t.Error("element has no id!", el)
		}
		compare.(ArbitraryMessage)["id"] = arb["id"]
	}
	if str, ok := el.(*StructMessage); ok {
		if id := str.MqId; id == "" {
			t.Error("Element has no id", el)
		}
		compare.(*StructMessage).MqId = str.MqId
	}

	if !reflect.DeepEqual(el, compare) {
		t.Error("List element isn't as it should be:", el, compare)
	}
}
