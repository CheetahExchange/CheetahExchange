package worker

import (
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/siddontang/go-log/log"
	"sync"
	"time"
)

var productsSupported sync.Map

func StartMatchingLogMaker() {
	spotConfig := conf.GetConfig()

	go func() {
		for {
			products, err := service.GetProducts()
			if err != nil {
				panic(err)
			}
			for _, product := range products {
				_, ok := productsSupported.Load(product.Id)
				if !ok {
					NewTickMaker(product.Id, matching.NewKafkaLogReader("tickMaker", product.Id, spotConfig.Kafka.Brokers)).Start()
					NewFillMaker(matching.NewKafkaLogReader("fillMaker", product.Id, spotConfig.Kafka.Brokers)).Start()
					NewTradeMaker(matching.NewKafkaLogReader("tradeMaker", product.Id, spotConfig.Kafka.Brokers)).Start()
					productsSupported.Store(product.Id, true)
					log.Infof("start maker for %s ok", product.Id)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

}
