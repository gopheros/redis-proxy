package main

import (
	"github.com/alicebob/miniredis"
	"github.com/gopheros/golang-lru"
	"github.com/magiconair/properties/assert"
	"testing"
	"time"
)



func NewTestRedisClient() (*lru.CacheWithTtl, RedisClient) {
	testConfig := GetConfig()
	mr, _ := miniredis.Run()
	testConfig.Set("redis.address", mr.Addr())
	testConfig.Set("cache.size", 2)
	testConfig.Set("cache.expiry", "1ms")
	logger := NewLogger()
	lru, _ := NewLruCache(logger, testConfig)
	testRedisClient := NewRedisClient(logger, testConfig)
	setUpRedisWithFakeData(logger, testRedisClient)

	testCacheBasedRedisClient := NewCacheBasedRedisClient(
		testRedisClient,
		lru,
		logger)
	return lru, testCacheBasedRedisClient
}

func TestCacheBasedRedisClient_Get(t *testing.T) {

	cache, testCacheBasedRedisClient := NewTestRedisClient()

	tests := []struct {
		name    string
		client  RedisClient
		cache   *lru.CacheWithTtl
		key    string
		want    string
		wantErr bool
		wantCacheHit bool
	}{
		{
			name: "Test fetch from backend success",
			client: testCacheBasedRedisClient,
			cache: cache,
			key: "a",
			want: "a",
			wantErr: false,
			wantCacheHit: false,
		},
		{
			name: "Test fetch from cache success",
			client: testCacheBasedRedisClient,
			cache: cache,
			key: "a",
			want: "a",
			wantErr: false,
			wantCacheHit: true,
		},
		{
			name: "Test fetch from backend fail",
			client: testCacheBasedRedisClient,
			cache: cache,
			key: "1",
			want: "",
			wantErr: true,
			wantCacheHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			cacheHit := tt.cache.Contains(tt.key)
			got, err := client.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v failed, got error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%v failed to get correct value, got = %v, want %v", tt.name, got, tt.want)
			}
			if cacheHit != tt.wantCacheHit {
				t.Errorf("%v has incorrect caching %v," +
					" cacheHit = %v, want %v",tt.name, tt.key, tt.cache.Contains(tt.key), tt.wantCacheHit)
			}
		})
	}

	time.Sleep(10 * time.Millisecond)

	// test cache expiry
	key := "a"
	cacheHit := cache.Contains(key)
	assert.Equal(t, cacheHit, false)
	got, err := testCacheBasedRedisClient.Get(key)
	assert.Equal(t, err, nil)
	assert.Equal(t, got, "a")
	val, ok := cache.Get("b")
	assert.Equal(t, ok, false)
	assert.Equal(t, val, nil)

	// test cache eviction
	testCacheBasedRedisClient.Get("b")
	testCacheBasedRedisClient.Get("c")
	assert.Equal(t, cache.Len(), 2)
	ok = cache.Contains("a")
	assert.Equal(t, ok, false)
	assert.Equal(t, val, nil)

}
