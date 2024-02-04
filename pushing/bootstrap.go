package pushing

import (
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/siddontang/go-log/log"
	"sync"
	"time"
)

var productsSupported sync.Map

func StartServer() {
	gbeConfig := conf.GetConfig()

	sub := newSubscription()

	newRedisStream(sub).Start()

	go func() {
		for {
			products, err := service.GetProducts()
			if err != nil {
				panic(err)
			}
			for _, product := range products {
				_, ok := productsSupported.Load(product.Id)
				if !ok {
					newTickerStream(product.Id, sub, matching.NewKafkaLogReader("tickerStream", product.Id, gbeConfig.Kafka.Brokers)).Start()
					newMatchStream(product.Id, sub, matching.NewKafkaLogReader("matchStream", product.Id, gbeConfig.Kafka.Brokers)).Start()
					newOrderBookStream(product.Id, sub, matching.NewKafkaLogReader("orderBookStream", product.Id, gbeConfig.Kafka.Brokers)).Start()
					productsSupported.Store(product.Id, true)
					log.Infof("start stream for %s ok", product.Id)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

	go NewServer(gbeConfig.PushServer.Addr, gbeConfig.PushServer.Path, sub).Run()

	log.Info("websocket server ok")
}
