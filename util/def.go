package util

import (
	"github.com/spf13/viper"
)

type ParmArgs struct {
	UploadFile   string
	DownloadFile string
	DeleteFile   string
	List         bool
}

var (
	P      ParmArgs
	Config *viper.Viper
)

const KeyRequestId = "requestId"
