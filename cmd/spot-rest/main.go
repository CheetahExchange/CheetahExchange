package main

import (
	"github.com/CheetahExchange/CheetahExchange/rest"
	_ "net/http/pprof"
)

func main() {
	rest.StartServer()
	select {}
}
