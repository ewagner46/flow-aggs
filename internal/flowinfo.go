package internal

import (
	"fmt"
	"strconv"
)

type flowInfo struct {
	SrcApp  *string `json:"src_app" db:"src_app"`
	DestApp *string `json:"dest_app" db:"dest_app"`
	VpcID   *string `json:"vpc_id" db:"vpc_id"`
	BytesTx *int    `json:"bytes_tx" db:"bytes_tx"`
	BytesRx *int    `json:"bytes_rx" db:"bytes_rx"`
	Hour    *int    `json:"hour" db:"hour"`
}

func (info *flowInfo) UniqueId() string {
	return *info.SrcApp + " " + *info.DestApp + " " + *info.VpcID + " " + strconv.Itoa(*info.Hour)
}

func (info *flowInfo) Add(add flowInfo) {
	*info.BytesTx += *add.BytesTx
	*info.BytesRx += *add.BytesRx
}

func (info *flowInfo) Print() {
	fmt.Printf("src_app:%s, dest_app:%s, vpc_id:%s, bytes_tx:%d, bytes_rx:%d, hour:%d\n",
		*info.SrcApp, *info.DestApp, *info.VpcID, *info.BytesTx, *info.BytesRx, *info.Hour)
}
