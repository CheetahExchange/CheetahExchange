package worker

import (
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"time"
)

var minutes = []int64{1, 3, 5, 15, 30, 60, 120, 240, 360, 720, 1440}

type TickMaker struct {
	ticks     map[int64]*models.Tick
	tickCh    chan models.Tick
	logReader matching.LogReader
	logOffset int64
	logSeq    int64
}

func NewTickMaker(productId string, logReader matching.LogReader) *TickMaker {
	t := &TickMaker{
		ticks:     map[int64]*models.Tick{},
		tickCh:    make(chan models.Tick, 1000),
		logReader: logReader,
	}

	// Load the latest tick recorded in the database
	for _, granularity := range minutes {
		tick, err := service.GetLastTickByProductId(productId, granularity)
		if err != nil {
			panic(err)
		}
		if tick != nil {
			log.Infof("load last tick: %v", tick)
			t.ticks[granularity] = tick
			t.logOffset = tick.LogOffset
			t.logSeq = tick.LogSeq
		}
	}

	t.logReader.RegisterObserver(t)
	return t
}

func (t *TickMaker) Start() {
	if t.logOffset > 0 {
		t.logOffset++
	}
	go t.logReader.Run(t.logSeq, t.logOffset)
	go t.flusher()
}

func (t *TickMaker) OnOpenLog(log *matching.OpenLog, offset int64) {
	// do nothing
}

func (t *TickMaker) OnDoneLog(log *matching.DoneLog, offset int64) {
	// do nothing
}

func (t *TickMaker) OnMatchLog(log *matching.MatchLog, offset int64) {
	for _, granularity := range minutes {
		tickTime := log.Time.UTC().Truncate(time.Duration(granularity) * time.Minute).Unix()

		tick, found := t.ticks[granularity]
		if !found || tick.Time != tickTime {
			tick = &models.Tick{
				Open:        log.Price,
				Close:       log.Price,
				Low:         log.Price,
				High:        log.Price,
				Volume:      log.Size,
				ProductId:   log.ProductId,
				Granularity: granularity,
				Time:        tickTime,
				LogOffset:   offset,
				LogSeq:      log.Sequence,
			}
			t.ticks[granularity] = tick
		} else {
			tick.Close = log.Price
			tick.Low = decimal.Min(tick.Low, log.Price)
			tick.High = decimal.Max(tick.High, log.Price)
			tick.Volume = tick.Volume.Add(log.Size)
			tick.LogOffset = offset
			tick.LogSeq = log.Sequence
		}

		t.tickCh <- *tick
	}
}

func (t *TickMaker) flusher() {
	var ticks []*models.Tick

	for {
		select {
		case tick := <-t.tickCh:
			ticks = append(ticks, &tick)

			if len(t.tickCh) > 0 && len(ticks) < 1000 {
				continue
			}

			for {
				err := service.AddTicks(ticks)
				if err != nil {
					log.Error(err)
					time.Sleep(time.Second)
					continue
				}
				ticks = nil
				break
			}
		}
	}
}
