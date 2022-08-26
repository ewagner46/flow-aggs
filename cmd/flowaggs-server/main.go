package main

import (
	"flowaggs/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

func readConfig() error {
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
func cleanup(embeddedDb internal.EmbeddedDb) {
	if err := embeddedDb.Stop(); err != nil {
		log.Fatalf("Could not stop database: %s.  Exiting...\n", err.Error())
	}
	log.Infof("Stopped database.")
}
func main() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
	if err := readConfig(); err != nil {
		log.Warningf("Could not read in config: %s. Proceeding with defaults...", err.Error())
	}

	log.Infof("Starting embedded database...")
	var embeddedDb internal.EmbeddedDb
	if err := embeddedDb.Start(); err != nil {
		log.Fatalf("Could not start database: %s. Exiting...", err.Error())
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func(embeddedDb internal.EmbeddedDb) {
		<-ch
		cleanup(embeddedDb)
		os.Exit(1)
	}(embeddedDb)

	log.Infof("Started embedded database.")

	var err error
	if err := embeddedDb.Connect(); err != nil {
		log.Fatalf("couldn't connect to the database: %s. Exiting...", err.Error())
	}
	log.Infof("Connected to the database.")

	workerPool, err := internal.CreateDbWorkerPool(embeddedDb)
	if err != nil {
		cleanup(embeddedDb)
		log.Fatalf("Failed to create database worker pool: %s. Exiting...", err.Error())
	}
	log.Infof("Created database worker pool.")

	if err = internal.InitRESTServer(&workerPool, embeddedDb); err != nil {
		log.Fatalf("Could not start REST server: %s. Exiting...", err.Error())
	}
}
