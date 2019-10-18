package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gopheros/golang-lru"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"log"
	"net/http"
	"os"
	"time"
)

type Cache struct {
	redisClient *redis.Client
	lru *lru.CacheWithTtl
	size int
	logger *log.Logger
}

func NewRedisClient(logger *log.Logger, viper *viper.Viper) *redis.Client {
	addr := viper.GetString("redis.address")
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	go setUpRedisWithFakeData(logger, client)
	return client
}

func setUpRedisWithFakeData(logger *log.Logger, client *redis.Client) {
	for _, c := range "abcedefghijklmnopqrstuvwxyz" {
		logger.Println("setting key", string(c))
		err := client.Set(string(c), string(c), 0).Err()
		if err != nil {
			logger.Print("error setting fake data", err)
			break
		}
	}
}

func NewCache(logger *log.Logger, viper *viper.Viper, redisClient *redis.Client) (*Cache, error) {
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
	return &Cache{
		redisClient: redisClient,
		lru: lru,
		size: size,
		logger: logger,
	}, nil
}

func (c *Cache) GetKey(key string) string {
	var value string
	var err error
	ok := c.lru.Contains(key)
	if !ok {
		c.logger.Printf("Cache miss!, key: %v not found in local cache \n", key)

		value, err = c.redisClient.Get(key).Result()
		if err != nil {
			c.logger.Printf( "Error retrieving key %v from backend redis \n", key, err)
			return "Key not found in backend"
		}

		c.logger.Printf("key: %v, value: %v retrieved from backend redis", key, value)

		c.logger.Printf("key: %v, value: %v added to local cache", key, value)
		evicted := c.lru.Add(key, value)
		if evicted {
			c.logger.Printf("eviction happened!")
		}
	} else {
		c.logger.Printf("key: %v found in local cache \n", key)
		v, _ := c.lru.Get(key);
		value = fmt.Sprintf("%v", v)
	}
	return value
}

func GetConfig() *viper.Viper {
	viper.SetConfigName("base")
	viper.AddConfigPath("config")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal("Config file not found")
		} else {
			log.Fatal( "Config file can't be read")
		}
	}
	return viper.GetViper()
}

func NewLogger() *log.Logger {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	logger.Print("Executing NewLogger.")
	return logger
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["key"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}

	key := keys[0]
	fmt.Println(key)
	// fetch from lru cache
}

func NewHandler(logger *log.Logger, viper *viper.Viper, cache *Cache) (http.Handler, error) {
	logger.Print("Executing NewHandler.")


	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Print("Got a request.")
		keys, ok := r.URL.Query()["key"]

		if !ok || len(keys[0]) < 1 {
			log.Println("Url Param 'key' is missing")
			return
		}

		key := keys[0]
		value := cache.GetKey(key)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	}), nil
}

func NewMux(lc fx.Lifecycle, logger *log.Logger, viper *viper.Viper) *http.ServeMux {
	logger.Print("Executing NewMux.")
	// First, we construct the mux and server. We don't want to start the server
	// until all handlers are registered.
	port := fmt.Sprintf(":%s", viper.GetString("server.port"))
	idleTimeout := time.Duration(viper.GetInt("server.idleTimeout"))
	mux := http.NewServeMux()
	server := &http.Server{
		Addr: port,
		IdleTimeout: idleTimeout,
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				logger.Print("Starting HTTP server.")
				err := server.ListenAndServe()
				if err != nil {
					logger.Fatal("Failed to start the server", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Print("Stopping HTTP server.")
			return server.Shutdown(ctx)
		},
	})

	return mux
}

func Register(mux *http.ServeMux, h http.Handler) {
	mux.Handle("/GET", h)
}


func main() {
	fmt.Println("redis proxy")

	fmt.Println( "reading config")
	viper := GetConfig()
	fmt.Println("Server config %v", viper.Get("server"))
	app := fx.New(
		// Provide all the constructors we need, which teaches Fx how we'd like to
		// construct the *log.Logger, http.Handler, and *http.ServeMux types.
		// Remember that constructors are called lazily, so this block doesn't do
		// much on its own.
		fx.Provide(
			NewLogger,
			NewHandler,
			GetConfig,
			NewCache,
			NewRedisClient,
			NewMux,
		),
		// Since constructors are called lazily, we need some invocations to
		// kick-start our application. In this case, we'll use Register. Since it
		// depends on an http.Handler and *http.ServeMux, calling it requires Fx
		// to build those types using the constructors above. Since we call
		// NewMux, we also register Lifecycle hooks to start and stop an HTTP
		// server.
		fx.Invoke(Register),
	)

	app.Run()
	<-app.Done()
}
