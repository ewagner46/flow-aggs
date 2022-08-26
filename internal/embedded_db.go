package internal

import (
	"database/sql"
	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

var schema = `
CREATE TABLE IF NOT EXISTS flows (
    src_app text,
    dest_app text,
    vpc_id text,
    bytes_tx integer,
    bytes_rx integer,
    hour     integer
);`

type EmbeddedDb struct {
	Db     *postgres.EmbeddedPostgres
	handle *sqlx.DB
}

func (db *EmbeddedDb) Connect() error {
	var err error
	db.handle, err = sqlx.Connect("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=flowaggs sslmode=disable")
	db.CreateSchemas()
	return err
}

func (db *EmbeddedDb) Start() error {
	db.Db = postgres.NewDatabase(postgres.DefaultConfig().
		RuntimePath(viper.GetString("db_runtime_location")).
		DataPath(viper.GetString("db_data_location")).
		BinariesPath(viper.GetString("db_binary_location")).
		Database("flowaggs"))
	err := db.Db.Start()

	if err != nil {
		return err
	}
	return nil
}

func (db *EmbeddedDb) Stop() error {
	err := db.Db.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (db *EmbeddedDb) CreateSchemas() {
	db.handle.MustExec(schema)
}

var FlowQuery = "SELECT * FROM flows WHERE src_app=$1 AND dest_app=$2 AND vpc_id=$3 AND hour=$4"

func (db *EmbeddedDb) WriteFlowToDb(info flowInfo) error {
	tx, err := db.handle.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			// propagate original error, ignore rollback error
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	var found flowInfo
	err = tx.Get(&found, FlowQuery, *info.SrcApp, *info.DestApp, *info.VpcID, *info.Hour)

	if err == sql.ErrNoRows {
		err = nil
		_, err := tx.Exec("INSERT INTO flows (src_app, dest_app, vpc_id, bytes_tx, bytes_rx, hour) VALUES ($1, $2, $3, $4, $5, $6)",
			*info.SrcApp, *info.DestApp, *info.VpcID, *info.BytesTx, *info.BytesRx, *info.Hour)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	} else {
		info.Add(found)
		_, err := tx.Exec("WITH subquery as ("+FlowQuery+") UPDATE flows SET bytes_tx=$5, bytes_rx=$6 FROM subquery WHERE subquery.src_app=flows.src_app",
			*info.SrcApp, *info.DestApp, *info.VpcID, *info.Hour, *info.BytesTx, *info.BytesRx)
		if err != nil {
			return err
		}

	}
	return err
}

func (db *EmbeddedDb) ReadFlowFromDb(info flowInfo) (*flowInfo, error) {
	var found flowInfo
	err := db.handle.Get(&found, FlowQuery, *info.SrcApp, *info.DestApp, *info.VpcID, *info.Hour)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &found, nil
}

func (db *EmbeddedDb) ReadHourFromDb(hour int) (*[]flowInfo, error) {
	var found []flowInfo
	err := db.handle.Select(&found, "SELECT * FROM flows WHERE hour=$1", hour)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &found, nil
}
