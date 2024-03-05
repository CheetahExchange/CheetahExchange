package matching

import (
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/siddontang/go-log/log"
	"sync"
	"time"
)

var productsSupported sync.Map

func StartEngine() {
	gbeConfig := conf.GetConfig()

	go func() {
		for {
			products, err := service.GetProducts()
			if err != nil {
				panic(err)
			}
			for _, product := range products {
				_, ok := productsSupported.Load(product.Id)
				if !ok {
					orderReader := NewKafkaOrderReader(product.Id, gbeConfig.Kafka.Brokers)
					snapshotStore := NewRedisSnapshotStore(product.Id)
					logStore := NewKafkaLogStore(product.Id, gbeConfig.Kafka.Brokers)
					matchEngine := NewEngine(product, orderReader, logStore, snapshotStore)
					matchEngine.Start()
					productsSupported.Store(product.Id, true)
					log.Infof("start matching engine for %s ok", product.Id)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

	log.Info("matching engine ok")
}
