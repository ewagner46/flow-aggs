package internal

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"hash/maphash"
	"time"
)

const MaxWorkers int = 256

type DbWorkerPool struct {
	workers    [MaxWorkers]DbWorker
	numWorkers uint64
	config     DbWorkerConfig
	seed       maphash.Seed
}

type DbWorker struct {
	ch     chan flowInfo
	config DbWorkerConfig
	id     uint64
	db     EmbeddedDb
}

type DbWorkerConfig struct {
	pollIntervalMs        int64
	batchMaxSize          int
	chanSize              int
	channelWriteTimeoutMs int64
}

// WriteFlowLogToWorker writes to a consistent worker based unique row key
// src_app + dest_app + vpc_id + hour so that message bursts can be batched into a
// single database update per key.  if the channel remains full too long, we're backed
// up on messages, probably waiting on a DB write

func (workerPool *DbWorkerPool) WriteFlowLogToWorker(info flowInfo) error {
	hashInput := info.UniqueId()
	workerIx := maphash.String(workerPool.seed, hashInput) % workerPool.numWorkers

	select {
	case workerPool.workers[workerIx].ch <- info:
	case <-time.After(time.Duration(workerPool.config.channelWriteTimeoutMs) * time.Millisecond):
		return fmt.Errorf("worker %d's buffer was full", workerIx)
	}
	return nil

}

func CreateDbWorkerPool(db EmbeddedDb) (DbWorkerPool, error) {
	var workerPool DbWorkerPool
	viper.SetDefault("db_num_workers", 16)
	workerPool.numWorkers = viper.GetUint64("db_num_workers")
	workerPool.config = getDbWorkerConfig()
	workerPool.seed = maphash.MakeSeed()

	for ix := uint64(0); ix < workerPool.numWorkers; ix++ {
		if err := workerPool.workers[ix].initialize(ix, workerPool.config, db); err != nil {
			return workerPool, err
		}
		go workerPool.workers[ix].batchAndWriteDbUpdates()
	}
	return workerPool, nil
}

func getDbWorkerConfig() DbWorkerConfig {
	var config DbWorkerConfig
	// see considerations on config parameter usage in config.yaml
	viper.SetDefault("db_worker_poll_interval_ms", 5)
	viper.SetDefault("db_worker_batch_size", 5000)
	viper.SetDefault("db_worker_chan_size", 5000)
	viper.SetDefault("db_worker_chan_write_timeout_ms", 250)

	config.pollIntervalMs = viper.GetInt64("db_worker_poll_interval_ms")
	config.channelWriteTimeoutMs = viper.GetInt64("db_worker_chan_write_timeout_ms")
	config.batchMaxSize = viper.GetInt("db_worker_batch_size")
	config.chanSize = viper.GetInt("db_worker_chan_size")

	return config
}
func (worker *DbWorker) initialize(id uint64, config DbWorkerConfig, db EmbeddedDb) error {
	worker.config = config
	worker.id = id
	worker.ch = make(chan flowInfo, config.chanSize)
	worker.db = db
	return nil
}

// Read from channel,
// Multiple workers keyed by hash on row modulo worker num
// write updates from post into channel (must handle timeout)
// instantiate db
// batch updates
// connect to db
//
// TODO
// combine batch with what's in database, write to database
// Handle get query (handle DB failure)
// Test code
// Update README with multi writer

func (worker *DbWorker) combineStoredAndNewFlowInfo(newFlowInfo flowInfo) {
	if err := worker.db.WriteFlowToDb(newFlowInfo); err != nil {
		// we can do error specific handling, but inside sql.Exec it looks like there is already
		// retry logic for transient failure cases. If all retries failed, let's emit error and move on
		log.Errorf("Error writing flow to database: %s\n", err.Error())
	}
}
func (worker *DbWorker) batchAndWriteDbUpdates() {
	for {
		batch := make(map[string]flowInfo)
		batchCount := 0

	batchLoop:
		for {
			select {
			case info, ok := <-worker.ch:
				if !ok {
					// channel closed
					return
				}
				key := info.UniqueId()
				if aggInfo, ok := batch[key]; ok {
					info.Add(aggInfo)
				}
				batch[key] = info

				batchCount += 1
				if batchCount == worker.config.batchMaxSize {
					// to avoid ending up in a non-batch state due to constant influx to the channel,
					// cap the batch size and let a new batch build up to get the aggregation benefit
					break batchLoop
				}
			default:
				// channel is empty, let a new batch build up
				break batchLoop
			}
		}
		if batchCount > 0 {
			log.Debugf("Batch updates after %d logs.\n", batchCount)
		}

		for _, info := range batch {
			worker.combineStoredAndNewFlowInfo(info)
		}
		sleep := time.Duration(worker.config.pollIntervalMs) * time.Millisecond
		time.Sleep(sleep)
	}
}
