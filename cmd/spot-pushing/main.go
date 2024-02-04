package main

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/pushing"
	_ "net/http/pprof"
)

func main() {
	go models.NewBinLogStream().Start()
	pushing.StartServer()
	select {}
}
