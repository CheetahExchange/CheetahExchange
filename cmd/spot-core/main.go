package main

import (
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/worker"
	_ "net/http/pprof"
)

func main() {
	worker.NewFillExecutor().Start()
	worker.NewBillExecutor().Start()

	matching.StartEngine()
	worker.StartMatchingLogMaker()
	select {}
}
