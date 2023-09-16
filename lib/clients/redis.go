package clients

import "github.com/redis/go-redis/v9"

// New creates a new redis client
// url: redis://<user>:<pass>@localhost:6379/<db>
func NewRedis(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(opt)
}
