package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// GetConfig reads the configuration provided for service
// and returns viper
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

// NewLogger constructs the logger for service
func NewLogger() *log.Logger {
	logger := log.New()
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetReportCaller(true)
	logger.SetLevel(log.InfoLevel)
	return logger
}

func main() {
	log.Info("redis proxy starting ...")

	app := fx.New(
		fx.Provide(
			NewLogger,
			GetHandler,
			GetConfig,
			NewLruCache,
			NewRedisClient,
			NewCacheBasedRedisClient,
			NewMux,
		),
		fx.Invoke(Register),
	)

	app.Run()
	<-app.Done()
}
