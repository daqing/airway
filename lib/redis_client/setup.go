package redis_client

import "github.com/redis/go-redis/v9"

var __RDB__ *redis.Client

func Setup(url string) {
	if __RDB__ != nil {
		return
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	__RDB__ = redis.NewClient(opts)
}

func RDB() *redis.Client {
	if __RDB__ == nil {
		panic("redis client not initialized")
	}

	return __RDB__
}
