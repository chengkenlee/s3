package util

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func usage() {
	fmt.Printf("\nUsage: %s [-c conf] [-h]\n\nOptions:\necs S3工具，支持上传、下载、删除、查看。\nauthor: chengken\n\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Println()
}

func init() {
	var (
		help        bool
		conf        string
		defaultConf string
	)

	execDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	defaultConf = fmt.Sprintf("%s/.%s.yaml", execDir, filepath.Base(os.Args[0]))

	flag.StringVar(&conf, "c", defaultConf, "conf file")
	flag.BoolVar(&P.List, "list", false, "s3 buckets list")
	flag.StringVar(&P.UploadFile, "upload", "", "upload filename")
	flag.StringVar(&P.DownloadFile, "download", "", "download filename")
	flag.StringVar(&P.DeleteFile, "delete", "", "delete filename")
	flag.BoolVar(&help, "h", false, "show help information")

	flag.Parse()
	flag.Usage = usage

	if help || len(P.DownloadFile) == 0 && len(P.UploadFile) == 0 && len(P.DeleteFile) == 0 && !P.List {
		flag.Usage()
		os.Exit(-1)
	}

	paths, name := filepath.Split(conf)
	Config = viper.New()
	Config.SetConfigFile(fmt.Sprintf("%s%s", paths, name))
}

func InitParStr(keyStr string) string {
	if err := Config.ReadInConfig(); err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return Config.GetString(keyStr)
}

func Parm() {
}
