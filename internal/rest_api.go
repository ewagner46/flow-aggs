package internal

import (
	"errors"
	"github.com/spf13/viper"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// all threads only write to the worker channels, which are thread safe
var workerPool *DbWorkerPool
var embeddedDb EmbeddedDb

func InitRESTServer(pool *DbWorkerPool, db EmbeddedDb) error {
	workerPool = pool
	embeddedDb = db
	router := gin.Default()
	if err := router.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		return err
	}
	router.POST("/flows", postFlowInfo)
	router.GET("/flows", getFlowInfoByHour)

	viper.SetDefault("rest_api_host", "localhost")
	viper.SetDefault("rest_api_port", 8080)

	if err := router.Run(viper.GetString("rest_api_host") + ":" + strconv.Itoa(viper.GetInt("rest_api_port"))); err != nil {
		return err
	}
	return nil
}

func parsePostJson(ctx *gin.Context) (*flowInfo, error) {
	var flow flowInfo
	if err := ctx.BindJSON(&flow); err != nil {
		return nil, errors.New("malformed json document for POST")
	}

	if flow.SrcApp == nil || flow.DestApp == nil || flow.VpcID == nil ||
		flow.BytesTx == nil || flow.BytesRx == nil || flow.Hour == nil {
		return nil, errors.New("missing one or more required JSON fields")
	}
	return &flow, nil
}
func postFlowInfo(ctx *gin.Context) {

	info, err := parsePostJson(ctx)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	err = workerPool.WriteFlowLogToWorker(*info)
	if err != nil {
		ctx.IndentedJSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	ctx.IndentedJSON(http.StatusCreated, info)

}

func parseHourArg(ctx *gin.Context) (int, error) {
	args := ctx.Request.URL.Query()
	var hour int
	hourVal, present := args["hour"]
	if present == false {
		return -1, errors.New("missing required argument 'hour'")
	}

	if len(hourVal) != 1 {
		return -1, errors.New("too many 'hour' args supplied")
	}

	hour, err := strconv.Atoi(hourVal[0])
	if err != nil {
		return -1, errors.New("malformed required argument 'hour'")
	}
	return hour, nil
}

func getFlowInfoByHour(ctx *gin.Context) {
	hour, err := parseHourArg(ctx)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	entries, err := embeddedDb.ReadHourFromDb(hour)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "error reading flow from db: " + err.Error()})
		return
	}
	ctx.IndentedJSON(http.StatusOK, entries)
}
