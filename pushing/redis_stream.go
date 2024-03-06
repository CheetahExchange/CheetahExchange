package pushing

import (
	"context"
	"encoding/json"
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/utils"
	"github.com/go-redis/redis/v8"
	"github.com/siddontang/go-log/log"
	"sync"
	"time"
)

type redisStream struct {
	sub   *subscription
	mutex sync.Mutex
}

func newRedisStream(sub *subscription) *redisStream {
	return &redisStream{
		sub:   sub,
		mutex: sync.Mutex{},
	}
}

func (s *redisStream) Start() {
	spotConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     spotConfig.Redis.Addr,
		Password: spotConfig.Redis.Password,
		DB:       0,
	})
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}

	go func() {
	symbolOrder:
		for {
			ctx := context.Background()
			ps := redisClient.Subscribe(ctx, models.TopicOrder)
			_, err := ps.Receive(ctx)
			if err != nil {
				log.Error(err)
				_ = ps.Unsubscribe(ctx, models.TopicOrder)
				continue symbolOrder
			}

			for {
				select {
				case msg := <-ps.Channel():
					var order models.Order
					err := json.Unmarshal([]byte(msg.Payload), &order)
					if err != nil {
						_ = ps.Unsubscribe(ctx, models.TopicOrder)
						continue symbolOrder
					}

					s.sub.publish(ChannelOrder.Format(order.ProductId, order.UserId), &OrderMessage{
						UserId:        order.UserId,
						Type:          "order",
						Sequence:      0,
						Id:            utils.I64ToA(order.Id),
						Price:         order.Price.String(),
						Size:          order.Size.String(),
						Funds:         "0",
						ProductId:     order.ProductId,
						Side:          order.Side.String(),
						OrderType:     order.Type.String(),
						CreatedAt:     order.CreatedAt.Format(time.RFC3339),
						FillFees:      order.FillFees.String(),
						FilledSize:    order.FilledSize.String(),
						ExecutedValue: order.ExecutedValue.String(),
						Status:        order.Status.String(),
						Settled:       order.Settled,
					})
				}
			}
		}
	}()

	go func() {
	symbolAccount:
		for {
			ctx := context.Background()
			ps := redisClient.Subscribe(ctx, models.TopicAccount)
			_, err := ps.Receive(ctx)
			if err != nil {
				log.Error(err)
				continue symbolAccount
			}

			for {
				select {
				case msg := <-ps.Channel():
					var account models.Account
					err := json.Unmarshal([]byte(msg.Payload), &account)
					if err != nil {
						continue symbolAccount
					}

					s.sub.publish(ChannelFunds.FormatWithUserId(account.UserId), &FundsMessage{
						Type:      "funds",
						Sequence:  0,
						UserId:    utils.I64ToA(account.UserId),
						Currency:  account.Currency,
						Hold:      account.Hold.String(),
						Available: account.Available.String(),
					})
				}
			}
		}
	}()

	go func() {
	symbolTrade:
		for {
			ctx := context.Background()
			ps := redisClient.Subscribe(ctx, models.TopicTrade)
			_, err := ps.Receive(ctx)
			if err != nil {
				log.Error(err)
				_ = ps.Unsubscribe(ctx, models.TopicTrade)
				continue symbolTrade
			}

			for {
				select {
				case msg := <-ps.Channel():
					var trade models.Trade
					err := json.Unmarshal([]byte(msg.Payload), &trade)
					if err != nil {
						_ = ps.Unsubscribe(ctx, models.TopicTrade)
						continue symbolTrade
					}

					// push to maker
					s.sub.publish(ChannelTrade.Format(trade.ProductId, trade.MakerUserId), &TradeMessage{
						Type:         "trade",
						Time:         trade.Time.Format(time.RFC3339),
						ProductId:    trade.ProductId,
						Price:        trade.Price.String(),
						Size:         trade.Size.String(),
						MakerOrderId: utils.I64ToA(trade.MakerOrderId),
						TakerOrderId: utils.I64ToA(trade.TakerOrderId),
						Side:         trade.Side.String(),
					})

					// push to taker
					s.sub.publish(ChannelTrade.Format(trade.ProductId, trade.TakerUserId), &TradeMessage{
						Type:         "trade",
						Time:         trade.Time.Format(time.RFC3339),
						ProductId:    trade.ProductId,
						Price:        trade.Price.String(),
						Size:         trade.Size.String(),
						MakerOrderId: utils.I64ToA(trade.MakerOrderId),
						TakerOrderId: utils.I64ToA(trade.TakerOrderId),
						Side:         trade.Side.String(),
					})
				}
			}
		}
	}()
}
