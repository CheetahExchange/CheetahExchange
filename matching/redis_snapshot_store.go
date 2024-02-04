package matching

import (
	"context"
	"encoding/json"
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/go-redis/redis/v8"
	"time"
)

const (
	topicSnapshotPrefix = "matching_snapshot_"
)

type RedisSnapshotStore struct {
	productId   string
	redisClient *redis.Client
}

func NewRedisSnapshotStore(productId string) SnapshotStore {
	gbeConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     gbeConfig.Redis.Addr,
		Password: gbeConfig.Redis.Password,
		DB:       0,
	})

	return &RedisSnapshotStore{
		productId:   productId,
		redisClient: redisClient,
	}
}

func (s *RedisSnapshotStore) Store(snapshot *Snapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return s.redisClient.Set(context.Background(), topicSnapshotPrefix+s.productId, buf, 7*24*time.Hour).Err()
}

func (s *RedisSnapshotStore) GetLatest() (*Snapshot, error) {
	ret, err := s.redisClient.Get(context.Background(), topicSnapshotPrefix+s.productId).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var snapshot Snapshot
	err = json.Unmarshal(ret, &snapshot)
	return &snapshot, err
}
