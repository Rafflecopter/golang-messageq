package messageq

import (
	"github.com/Rafflecopter/golang-messageq/discovery/redis"
	"github.com/garyburd/redigo/redis"
)

func NewRedis(pool *redis.Pool, cfg *Config, discoveryPrefix string) *MessageQueue {
	cfg.Defaults()
	disco := redisdisco.New(pool, discoveryPrefix, cfg.Delimiter)
	return New(pool, disco, cfg)
}
