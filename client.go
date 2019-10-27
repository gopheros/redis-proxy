package main

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gopheros/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// RedisClient interface for redis client
type RedisClient interface {
	Get(key string) (string, error)
}

// CacheBasedRedisClient struct for implementing cache based redis client
type CacheBasedRedisClient struct {
	redisClient *redis.Client
	cache *lru.CacheWithTtl
	logger *log.Logger
}

// NewCacheBasedRedisClient constructs a cache based redis client for the proxy
func NewCacheBasedRedisClient(
	redisClient *redis.Client,
	cache *lru.CacheWithTtl,
	logger *log.Logger) RedisClient {
	return &CacheBasedRedisClient{
		redisClient: redisClient,
		cache: cache,
		logger: logger,
	}
}

// Get implements redis GET with additional caching layer
func (client *CacheBasedRedisClient) Get(key string) (string, error) {
	var val interface{}
	var err error
	var ok bool

	if !client.cache.Contains(key) {
		val, err = client.redisClient.Get(key).Result()
		if err != nil {
			msg := fmt.Sprintf("error while fetching key %v from backend \n", key)
			client.logger.Error(msg)
			return "", err
		}
		client.cache.Add(key, fmt.Sprintf("%v", val))
		client.logger.Infof("succesfully fetched key %v from backend \n", key)
	} else {
		val, ok = client.cache.Get(key)
		if val == nil || !ok {
			msg := fmt.Sprintf("error while fetching key %v from cache \n", key)
			client.logger.Errorf(msg)
			return "", errors.New(msg)
		}
		client.logger.Infof("succesfully fetched key %v from cache \n", key)
	}
	return fmt.Sprintf("%v", val), nil
}

/*--------------------------------------------------------------------------------------------------------------------*/

// NewRedisClient constructs the redis client to backend redis
func NewRedisClient(logger *log.Logger, viper *viper.Viper) *redis.Client {
	addr := viper.GetString("redis.address")
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return client
}

// NewLruCache constructs the lru cache used by proxy
func NewLruCache(logger *log.Logger, viper *viper.Viper) (*lru.CacheWithTtl, error) {
	size := viper.GetInt("cache.size")
	expiry, err := time.ParseDuration(viper.GetString("cache.expiry"))
	if err != nil {
		logger.Fatal("Error parsing expiry", err)
		return nil, err
	}
	lru, err := lru.NewTtl(size, expiry)
	if err != nil {
		logger.Fatal("Error creating local cache", err)
		return nil, err
	}
	return lru, nil
}

func setUpRedisWithFakeData(logger *log.Logger, client *redis.Client) {
	for _, c := range "abcdefghijklmnopqrstuvwxyz" {
		err := client.Set(string(c), string(c), 0).Err()
		if err != nil {
			logger.Print("error setting fake data", err)
			break
		}
	}
}