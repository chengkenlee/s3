package main

import (
	"s3/service"
	"s3/util"
)

func main() {
	util.Parm()
	util.InitParStr("")
	service.Run()
}
