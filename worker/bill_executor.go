package worker

import (
	"context"
	"encoding/json"
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/go-redis/redis/v8"
	"github.com/siddontang/go-log/log"
	"time"
)

const billWorkerNum = 10

type BillExecutor struct {
	workerChs [billWorkerNum]chan *models.Bill
}

func NewBillExecutor() *BillExecutor {
	f := &BillExecutor{
		workerChs: [billWorkerNum]chan *models.Bill{},
	}

	// Initialize the same number of routines as billWorkerNum, each responsible for one chan.
	for i := 0; i < billWorkerNum; i++ {
		f.workerChs[i] = make(chan *models.Bill, 256)
		go func(idx int) {
			for {
				select {
				case bill := <-f.workerChs[idx]:
					err := service.ExecuteBill(bill.UserId, bill.Currency)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}(i)
	}
	return f
}

func (s *BillExecutor) Start() {
	go s.runMqListener()
	go s.runInspector()
}

func (s *BillExecutor) runMqListener() {
	spotConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     spotConfig.Redis.Addr,
		Password: spotConfig.Redis.Password,
		DB:       0,
	})

	for {
		ret := redisClient.BRPop(context.Background(), time.Second*1000, models.TopicBill)
		if ret.Err() != nil {
			log.Error(ret.Err())
			continue
		}

		var bill models.Bill
		err := json.Unmarshal([]byte(ret.Val()[1]), &bill)
		if err != nil {
			panic(ret.Err())
		}

		// Sharding by userId
		s.workerChs[bill.UserId%billWorkerNum] <- &bill
	}
}

func (s *BillExecutor) runInspector() {
	for {
		select {
		case <-time.After(1 * time.Second):
			bills, err := service.GetUnsettledBills()
			if err != nil {
				log.Error(err)
				continue
			}

			for _, bill := range bills {
				s.workerChs[bill.UserId%billWorkerNum] <- bill
			}
		}
	}
}
