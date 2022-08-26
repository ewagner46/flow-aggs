package internal

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// all threads only write to the worker channels, which are thread safe
var workerPool *DbWorkerPool

type flowInfo struct {
	SrcApp  *string `json:"src_app"`
	DestApp *string `json:"dest_app"`
	VpcID   *string `json:"vpc_id"`
	BytesTx *int    `json:"bytes_tx"`
	BytesRx *int    `json:"bytes_rx"`
	Hour    *int    `json:"hour"`
}

func InitRESTServer(pool *DbWorkerPool) error {
	workerPool = pool
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})
	router.POST("/flows", postFlowInfo)
	router.GET("/flows", getFlowInfoByHour)

	if err := router.Run("localhost:8080"); err != nil {
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
	}

	fmt.Printf("receive %s\n", *info.SrcApp)
	//fmt.Printf("post receive src_app:%s\ndest_app:%s\n,vpc_id:%s\nbytes_tx:%d\nbytes_rx:%d\nhour:%d\n",
	//	*info.SrcApp, *info.DestApp, *info.VpcID, *info.BytesTx, *info.BytesRx, *info.Hour)

	ctx.IndentedJSON(http.StatusCreated, info)

}

func parseHourArg(ctx *gin.Context) (int, error) {
	args := ctx.Request.URL.Query()
	var hour int
	hourVals, present := args["hour"]
	if present == false {
		return -1, errors.New("missing required argument 'hour'")
	}

	if len(hourVals) != 1 {
		return -1, errors.New("too many 'hour' args supplied")
	}

	hour, err := strconv.Atoi(hourVals[0])
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

	fmt.Printf("hour=%d\n", hour)
	ctx.IndentedJSON(http.StatusOK, nil)
}
