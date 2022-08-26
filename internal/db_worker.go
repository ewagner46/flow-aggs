package internal

import (
	"fmt"
	"github.com/spf13/viper"
	"hash/fnv"
	"strconv"
	"time"
)

const MaxWorkers int = 256

type DbWorkerPool struct {
	workers    [MaxWorkers]DbWorker
	numWorkers uint32
	config     DbWorkerConfig
}

type DbWorker struct {
	ch     chan flowInfo
	config *DbWorkerConfig
}

type DbWorkerConfig struct {
	pollIntervalMs        int64
	batchMaxSize          int
	chanSize              int
	channelWriteTimeoutMs int
}

func hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

// WriteFlowLogToWorker writes to a consistent worker based unique row key
// src_app + dest_app + vpc_id + hour so that message bursts can be batched into a
// single database update per key.  if the channel remains full too long, we're backed
// up on messages, probably waiting on a DB write
func WriteFlowLogToWorker(workerPool DbWorkerPool, info flowInfo) error {
	hashKey := *info.SrcApp + " " + *info.DestApp + " " + *info.VpcID + " " + strconv.Itoa(*info.Hour)
	workerIx, err := hash(hashKey)
	if err != nil {
		return fmt.Errorf("error while hashing flow key: %s", err.Error())
	}
	workerIx = workerIx % workerPool.numWorkers

	fmt.Printf("write %s to worker %d\n", hashKey, workerIx)
	select {
	case workerPool.workers[workerIx].ch <- info:
	case <-time.After(time.Duration(workerPool.config.channelWriteTimeoutMs) * time.Millisecond):
		return fmt.Errorf("worker %d's buffer was full", workerIx)
	}
	return nil

}

func CreateDbWorkerPool() DbWorkerPool {
	var workerPool DbWorkerPool
	viper.SetDefault("db_num_workers", 16)
	workerPool.numWorkers = viper.GetUint32("db_num_workers")
	workerPool.config = getDbWorkerConfig()
	for ix := uint32(0); ix < workerPool.numWorkers; ix++ {
		workerPool.workers[ix] = createDbWorker(&workerPool.config)
		go batchAndDbWrite(workerPool.workers[ix])
	}
	return workerPool
}

func getDbWorkerConfig() DbWorkerConfig {
	var config DbWorkerConfig
	// see considerations on config parameter usage in config.yaml
	viper.SetDefault("db_worker_poll_interval_ms", 5)
	viper.SetDefault("db_worker_batch_size", 5000)
	viper.SetDefault("db_worker_chan_size", 5000)

	config.pollIntervalMs = viper.GetInt64("db_worker_poll_interval_ms")
	config.batchMaxSize = viper.GetInt("db_worker_batch_size")
	config.chanSize = viper.GetInt("db_worker_chan_size")

	return config
}
func createDbWorker(config *DbWorkerConfig) DbWorker {
	var dbWorker DbWorker
	dbWorker.config = config
	dbWorker.ch = make(chan flowInfo, config.chanSize)
	return dbWorker
}

// Read from channel,
// Multiple workers keyed by hash on row modulo worker num
// write updates from post into channel (must handle timeout)
// instantiate db
//
// TODO
// connect to db
// batch updates
// combine batch with what's in database, write to database
// handle db failure
// Handle get query (handle DB failure)
// Test code
//
//	Update README with multi writer
func batchAndDbWrite(worker DbWorker) {
	for {
	batchLoop:
		for {
			batch := 0
			select {
			case info, ok := <-worker.ch:
				fmt.Printf("read %s\n", *info.SrcApp)
				//fmt.Printf("read channel src_app:%s\ndest_app:%s\n,vpc_id:%s\nbytes_tx:%d\nbytes_rx:%d\nhour:%d\n",
				//	*info.SrcApp, *info.DestApp, *info.VpcID, *info.BytesTx, *info.BytesRx, *info.Hour)
				if !ok {
					// channel closed
					return
				}
				batch += 1
				if batch == worker.config.batchMaxSize {
					// to avoid ending up in a non-batch state due to constant influx to the channel,
					// cap the batch size and let a new batch build up to get the aggregation benefit
					break batchLoop
				}
			default:
				// channel is empty, let a new batch build up
				break batchLoop
			}
		}
		sleep := time.Duration(worker.config.pollIntervalMs) * time.Millisecond
		time.Sleep(sleep)
	}
}
