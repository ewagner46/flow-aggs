package internal

import (
	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/spf13/viper"
)

type EmbeddedDb struct {
	Db *postgres.EmbeddedPostgres
}

func StartDb() (*EmbeddedDb, error) {
	var db EmbeddedDb
	db.Db = postgres.NewDatabase(postgres.DefaultConfig().
		RuntimePath(viper.GetString("db_runtime_location")).
		DataPath(viper.GetString("db_data_location")).
		BinariesPath(viper.GetString("db_binary_location")))
	err := db.Db.Start()

	if err != nil {
		return nil, err
	}
	return &db, nil
}

func StopDb(db *EmbeddedDb) error {
	err := db.Db.Stop()
	if err != nil {
		return err
	}
	return nil
}
