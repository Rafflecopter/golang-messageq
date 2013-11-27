// Package redisdisco provides a redis-backed discovery service for messageq.
package redisdisco

import (
	"github.com/garyburd/redigo/redis"
)

type RedisDiscovery struct {
	pool   *redis.Pool
	prefix string
}

func New(pool *redis.Pool, prefix, delim string) *RedisDiscovery {
	return &RedisDiscovery{
		pool:   pool,
		prefix: prefix + delim,
	}
}

func (rd *RedisDiscovery) Register(channel, endpoint string) error {
	key := rd.prefix + channel
	_, err := rd.do("SADD", key, endpoint)

	return err
}

func (rd *RedisDiscovery) Unregister(channel, endpoint string) error {
	key := rd.prefix + channel
	_, err := rd.do("SREM", key, endpoint)
	return err
}

func (rd *RedisDiscovery) Subscribers(channel string) ([]string, error) {
	key := rd.prefix + channel
	list, err := redis.Values(rd.do("SMEMBERS", key))
	if err != nil {
		return nil, err
	}

	slist := make([]string, 0, len(list))
	for _, el := range list {
		if s, ok := el.([]byte); ok {
			slist = append(slist, string(s))
		}
	}

	return slist, nil
}

func (rd *RedisDiscovery) Close() error {
	return nil
}

func (rd *RedisDiscovery) do(cmd string, args ...interface{}) (interface{}, error) {
	conn := rd.pool.Get()
	defer conn.Close()
	return conn.Do(cmd, args...)
}
