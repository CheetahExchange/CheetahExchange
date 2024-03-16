package pushing

import (
	"fmt"
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/shopspring/decimal"
	logger "github.com/siddontang/go-log/log"
	"sync"
	"time"
)

const (
	//intervalSec = 3
	intervalSec = 1
)

var (
	lastTickers = sync.Map{}
)

type TickerStream struct {
	productId       string
	sub             *subscription
	bestBid         decimal.Decimal
	bestAsk         decimal.Decimal
	logReader       matching.LogReader
	lastTickerTime  int64
	lastCandlesTime int64
}

func newTickerStream(productId string, sub *subscription, logReader matching.LogReader) *TickerStream {
	s := &TickerStream{
		productId:      productId,
		sub:            sub,
		logReader:      logReader,
		lastTickerTime: time.Now().Unix() - intervalSec,
	}
	s.logReader.RegisterObserver(s)
	return s
}

func (s *TickerStream) Start() {
	// -1 : read from end
	go s.logReader.Run(0, -1)
}

func (s *TickerStream) OnOpenLog(log *matching.OpenLog, offset int64) {
	// do nothing
}

func (s *TickerStream) OnDoneLog(log *matching.DoneLog, offset int64) {
	// do nothing
}

func (s *TickerStream) OnMatchLog(log *matching.MatchLog, offset int64) {
	if (time.Now().Unix() - s.lastTickerTime) > intervalSec {
		ticker, err := s.newTickerMessage(log)
		if err != nil {
			logger.Error(err)
			return
		}
		if ticker == nil {
			return
		}
		lastTickers.Store(log.ProductId, ticker)
		s.sub.publish(ChannelTicker.FormatWithProductId(log.ProductId), ticker)
		s.lastTickerTime = time.Now().Unix()
	} else {
		ticker := getLastTicker(log.ProductId)
		if ticker == nil {
			return
		}
		ticker.TradeSeq = log.TradeSeq
		ticker.Sequence = log.Sequence
		ticker.Time = log.Time.Format(time.RFC3339)
		ticker.ProductId = log.ProductId
		ticker.Price = log.Price.String()
		ticker.Side = log.Side.String()
		ticker.LastSize = log.Size.String()
		lastTickers.Store(log.ProductId, ticker)
		s.sub.publish(ChannelTicker.FormatWithProductId(log.ProductId), ticker)
	}

	// publish candles info one second in the future
	if (time.Now().Unix() - s.lastCandlesTime) > intervalSec {
		go func(delaySec int64, productId string) {
			select {
			case <-time.After(time.Duration(delaySec) * time.Second):
				ticks, err := service.GetLastTicksAllByProductId(productId)
				if err != nil {
					return
				}
				for _, tick := range ticks {
					candles := s.newCandlesMessage(tick.Granularity, productId, tick)
					s.sub.publish(CandlesFormatWithGranularityProductId(tick.Granularity, productId), candles)
				}
			}
		}(1, log.ProductId)
		s.lastCandlesTime = time.Now().Unix()
	}
}

func (s *TickerStream) newTickerMessage(log *matching.MatchLog) (*TickerMessage, error) {
	ticks24h, err := service.GetTicksByProductId(s.productId, 1*60, 0, 0, 24)
	if err != nil {
		return nil, err
	}
	tick24h := mergeTicks(ticks24h)
	if tick24h == nil {
		tick24h = &models.Tick{}
	}

	ticks30d, err := service.GetTicksByProductId(s.productId, 24*60, 0, 0, 30)
	if err != nil {
		return nil, err
	}
	tick30d := mergeTicks(ticks30d)
	if tick30d == nil {
		tick30d = &models.Tick{}
	}

	return &TickerMessage{
		Type:      "ticker",
		TradeSeq:  log.TradeSeq,
		Sequence:  log.Sequence,
		Time:      log.Time.Format(time.RFC3339),
		ProductId: log.ProductId,
		Price:     log.Price.String(),
		Side:      log.Side.String(),
		LastSize:  log.Size.String(),
		Open24h:   tick24h.Open.String(),
		Low24h:    tick24h.Low.String(),
		Volume24h: tick24h.Volume.String(),
		Volume30d: tick30d.Volume.String(),
	}, nil
}

func (s *TickerStream) newCandlesMessage(granularity int64, productId string, tick *models.Tick) *CandlesMessage {
	return &CandlesMessage{
		Type:      fmt.Sprintf("candles_%dm", granularity),
		ProductId: productId,
		Time:      time.Unix(tick.Time, 0).Format(time.RFC3339),
		Open:      tick.Open.String(),
		Close:     tick.Close.String(),
		Low:       tick.Low.String(),
		High:      tick.High.String(),
		Volume:    tick.Volume.String(),
	}
}

func mergeTicks(ticks []*models.Tick) *models.Tick {
	var t *models.Tick
	for i := range ticks {
		tick := ticks[len(ticks)-1-i]
		if t == nil {
			t = tick
		} else {
			t.Open = tick.Open
			t.Close = tick.Close
			t.Low = decimal.Min(t.Low, tick.Low)
			t.High = decimal.Max(t.High, tick.High)
			t.Volume = t.Volume.Add(tick.Volume)
		}
	}
	return t
}

func getLastTicker(productId string) *TickerMessage {
	ticker, found := lastTickers.Load(productId)
	if !found {
		return nil
	}
	return ticker.(*TickerMessage)
}
