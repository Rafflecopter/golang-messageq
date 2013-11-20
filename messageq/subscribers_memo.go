package messageq

import (
	"time"
)

type subscriberser interface {
	Subscribers(string) ([]string, error)
}

// A decaying memoization of the S.Subscribers() function
type subscribersMemo struct {
	S     subscriberser
	M     map[string]*decayingSubscribers
	Decay time.Duration
}

type decayingSubscribers struct {
	S []string
	T time.Time
}

func memoizeSubscriberser(s subscriberser, decay time.Duration) *subscribersMemo {
	return &subscribersMemo{
		S:     s,
		M:     make(map[string]*decayingSubscribers),
		Decay: decay,
	}
}

func (sm *subscribersMemo) Subscribers(x string) ([]string, error) {
	if ds, ok := sm.M[x]; ok && !ds.Decayed(sm.Decay) {
		return ds.S, nil
	}

	s, err := sm.S.Subscribers(x)
	if err != nil {
		return nil, err
	}

	sm.M[x] = newDecayingSubscribers(s)
	return s, nil
}

func newDecayingSubscribers(s []string) *decayingSubscribers {
	return &decayingSubscribers{s, time.Now()}
}

func (ds *decayingSubscribers) Decayed(d time.Duration) bool {
	return ds.T.Add(d).Sub(time.Now()) < 0
}
