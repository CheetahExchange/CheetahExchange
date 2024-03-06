package rest

import (
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/siddontang/go-log/log"
)

func StartServer() {
	spotConfig := conf.GetConfig()

	httpServer := NewHttpServer(spotConfig.RestServer.Addr)
	go httpServer.Start()

	log.Info("rest server ok")
}
