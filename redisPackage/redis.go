package redisPackage

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/go-redis/redis/v9"
)

type RedisInterface struct {
	Client *redis.Client
}

func NewRedisInterface(redisURI string) (*RedisInterface, error) {
	opt, err := redis.ParseURL(redisURI)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	_, err = client.FlushAll(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &RedisInterface{
		Client: client,
	}, nil

}

func (r *RedisInterface) HSetObj(key, field string, value any, expiration time.Duration) (err error) {
	marshalledValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = r.Client.HSet(context.Background(), key, field, marshalledValue).Err()
	if err != nil {
		return err
	}

	return r.Client.Expire(context.Background(), key, expiration).Err()
}

func (r *RedisInterface) GetObj(key string, obj any) (err error) {
	value, err := r.Client.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(value), &obj)
}
