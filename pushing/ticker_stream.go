package pushing

import (
	"fmt"
	"sync"
	"time"

	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/shopspring/decimal"
	logger "github.com/siddontang/go-log/log"
)

const (
	//intervalSec = 3
	intervalSec = 1
)

var (
	lastTickers = sync.Map{}
)

type tickerStats struct {
	Open24h   decimal.Decimal
	Low24h    decimal.Decimal
	High24h   decimal.Decimal
	Volume24h decimal.Decimal
	Volume30d decimal.Decimal
	loaded    bool
	lastSync  time.Time
	mu        sync.RWMutex
}

type TickerStream struct {
	productId       string
	sub             *subscription
	bestBid         decimal.Decimal
	bestAsk         decimal.Decimal
	logReader       matching.LogReader
	lastTickerTime  int64
	lastCandlesTime int64
	stats           *tickerStats
}

func newTickerStream(productId string, sub *subscription, logReader matching.LogReader) *TickerStream {
	s := &TickerStream{
		productId:      productId,
		sub:            sub,
		logReader:      logReader,
		lastTickerTime: time.Now().Unix() - intervalSec,
		stats:          &tickerStats{},
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
	s.updateTickerStats(log)

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
			time.Sleep(time.Duration(delaySec) * time.Second)
			ticks, err := service.GetLastTicksAllByProductId(productId)
			if err != nil {
				return
			}
			for _, tick := range ticks {
				candles := s.newCandlesMessage(tick.Granularity, productId, tick)
				s.sub.publish(CandlesFormatWithGranularityProductId(tick.Granularity, productId), candles)
			}
		}(1, log.ProductId)
		s.lastCandlesTime = time.Now().Unix()
	}
}

func (s *TickerStream) loadTickerStats() {
	ticks24h, err := service.GetTicksByProductId(s.productId, 1*60, 0, 0, 24)
	if err != nil {
		logger.Error(err)
		return
	}
	tick24h := mergeIntervalTicks(ticks24h, 24*3600)
	if tick24h == nil {
		tick24h = &models.Tick{}
	}

	ticks30d, err := service.GetTicksByProductId(s.productId, 24*60, 0, 0, 30)
	if err != nil {
		logger.Error(err)
		return
	}
	tick30d := mergeIntervalTicks(ticks30d, 30*86400)
	if tick30d == nil {
		tick30d = &models.Tick{}
	}

	s.stats.mu.Lock()
	s.stats.Open24h = tick24h.Open
	s.stats.Low24h = tick24h.Low
	s.stats.High24h = tick24h.High
	s.stats.Volume24h = tick24h.Volume
	s.stats.Volume30d = tick30d.Volume
	s.stats.loaded = true
	s.stats.lastSync = time.Now()
	s.stats.mu.Unlock()
}

func (s *TickerStream) updateTickerStats(log *matching.MatchLog) {
	s.stats.mu.RLock()
	loaded := s.stats.loaded
	lastSync := s.stats.lastSync
	s.stats.mu.RUnlock()

	if !loaded {
		s.loadTickerStats()
		return
	}

	if time.Since(lastSync) > 5*time.Minute {
		go s.loadTickerStats()
	}

	s.stats.mu.Lock()
	if s.stats.Low24h.IsZero() || log.Price.LessThan(s.stats.Low24h) {
		s.stats.Low24h = log.Price
	}
	if log.Price.GreaterThan(s.stats.High24h) {
		s.stats.High24h = log.Price
	}
	s.stats.Volume24h = s.stats.Volume24h.Add(log.Size)
	s.stats.Volume30d = s.stats.Volume30d.Add(log.Size)
	s.stats.mu.Unlock()
}

func (s *TickerStream) newTickerMessage(log *matching.MatchLog) (*TickerMessage, error) {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return &TickerMessage{
		Type:      "ticker",
		TradeSeq:  log.TradeSeq,
		Sequence:  log.Sequence,
		Time:      log.Time.Format(time.RFC3339),
		ProductId: log.ProductId,
		Price:     log.Price.String(),
		Side:      log.Side.String(),
		LastSize:  log.Size.String(),
		Open24h:   s.stats.Open24h.String(),
		Low24h:    s.stats.Low24h.String(),
		Volume24h: s.stats.Volume24h.String(),
		Volume30d: s.stats.Volume30d.String(),
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

func mergeIntervalTicks(ticks []*models.Tick, interval int64) *models.Tick {
	var t *models.Tick
	for i := range ticks {
		tick := ticks[len(ticks)-1-i]
		if tick.Time < (time.Now().Unix() - interval) {
			continue
		}
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