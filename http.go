package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"net/http"
	"time"
)

func bindMaxClients(h http.Handler, n int) http.Handler {
	sema := make(chan struct{}, n)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sema <- struct{}{}
		defer func() { <-sema }()

		h.ServeHTTP(w, r)
	})
}

// GetHandler constructs the handler for /GET http API
func GetHandler(logger *log.Logger, client RedisClient) http.Handler {
	logger.Info("Executing Get Handler.")

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("Got a GET request.")
		keys, ok := r.URL.Query()["key"]

		if !ok || len(keys[0]) < 1 {
			log.Error("Url Param 'key' is missing")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		key := keys[0]
		value, err := client.Get(key)
		if err != nil {
			value = "key not found"
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	})
	maxClients := viper.GetInt("server.maxClients")
	return bindMaxClients(h, maxClients)
}


// NewMux constructs http server, http mux, addes fx lifecycle hooks
func NewMux(lc fx.Lifecycle, logger *log.Logger, viper *viper.Viper) *http.ServeMux {
	logger.Debug("Executing NewMux.")

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
				logger.Infof("Starting HTTP server on port %v.", port)
				err := server.ListenAndServe()
				if err != nil {
					logger.Fatal("Failed to start the server", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server.")
			return server.Shutdown(ctx)
		},
	})

	return mux
}

// Register registers the handler for http server
func Register(mux *http.ServeMux, h http.Handler) {
	mux.Handle("/GET", h)
}
