package worker

import (
	"context"
	"encoding/json"
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/go-redis/redis/v8"
	lru "github.com/hashicorp/golang-lru"
	"github.com/siddontang/go-log/log"
	"time"
)

const fillWorkerNum = 10

type FillExecutor struct {
	// Used to receive the fill after sharding, and sharding by orderId can avoid lock contention.
	workerChs [fillWorkerNum]chan *models.Fill
}

func NewFillExecutor() *FillExecutor {
	f := &FillExecutor{
		workerChs: [fillWorkerNum]chan *models.Fill{},
	}

	// Initialize as many routines as fillWorkerNum, each responsible for one chan.
	for i := 0; i < fillWorkerNum; i++ {
		f.workerChs[i] = make(chan *models.Fill, 512)
		go func(idx int) {
			settledOrderCache, err := lru.New(1000)
			if err != nil {
				panic(err)
			}

			for {
				select {
				case fill := <-f.workerChs[idx]:
					if settledOrderCache.Contains(fill.OrderId) {
						continue
					}

					order, err := service.GetOrderById(fill.OrderId)
					if err != nil {
						log.Error(err)
					}
					if order == nil {
						log.Warnf("order not found: %v", fill.OrderId)
						continue
					}

					// completed order
					if order.Status == models.OrderStatusCancelled || order.Status == models.OrderStatusFilled ||
						order.Status == models.OrderStatusPartial {
						settledOrderCache.Add(order.Id, struct{}{})
						continue
					}

					err = service.ExecuteFill(fill.OrderId)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}(i)
	}

	return f
}

func (s *FillExecutor) Start() {
	go s.runInspector()
	go s.runMqListener()
}

// Listening for message queue notifications
func (s *FillExecutor) runMqListener() {
	spotConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     spotConfig.Redis.Addr,
		Password: spotConfig.Redis.Password,
		DB:       0,
	})

	for {
		ret := redisClient.BRPop(context.Background(), time.Second*1000, models.TopicFill)
		if ret.Err() != nil {
			log.Error(ret.Err())
			continue
		}

		var fill models.Fill
		err := json.Unmarshal([]byte(ret.Val()[1]), &fill)
		if err != nil {
			log.Error(err)
			continue
		}

		// Sharding is performed by taking a pattern from the orderId, the same orderId will be assigned to the same chan
		s.workerChs[fill.OrderId%fillWorkerNum] <- &fill
	}
}

// Timed Polling Database
func (s *FillExecutor) runInspector() {
	for {
		select {
		case <-time.After(1 * time.Second):
			fills, err := service.GetUnsettledFills(1000)
			if err != nil {
				log.Error(err)
				continue
			}

			for _, fill := range fills {
				s.workerChs[fill.OrderId%fillWorkerNum] <- fill
			}
		}
	}
}
