package main

import (
	"flowaggs/internal"
	"github.com/spf13/viper"
	"log"
)

func main() {
	// see considerations on config parameter usage in config.yaml
	viper.SetDefault("db_worker_poll_interval_ms", 5)
	viper.SetDefault("db_worker_batch_size", 5000)
	viper.SetDefault("db_worker_chan_size", 5000):
	err := internal.InitRESTServer()
	if err != nil {
		log.Fatalf("Could not start REST server: %s. Exiting...", err.Error())
	}
}
