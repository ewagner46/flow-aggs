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
func cleanup(embeddedDb *internal.EmbeddedDb) {
	if err := internal.StopDb(embeddedDb); err != nil {
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
	embeddedDb, err := internal.StartDb()
	if err != nil {
		log.Fatalf("Could not start database: %s. Exiting...", err.Error())
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func(embeddedDb *internal.EmbeddedDb) {
		<-ch
		cleanup(embeddedDb)
		os.Exit(1)
	}(embeddedDb)

	log.Infof("Started embedded database.")

	workerPool := internal.CreateDbWorkerPool()
	log.Infof("Created database worker pool.")

	err = internal.InitRESTServer(&workerPool)
	if err != nil {
		log.Fatalf("Could not start REST server: %s. Exiting...", err.Error())
	}
}
