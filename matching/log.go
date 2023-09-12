// Copyright 2019 GitBitEx.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package matching

import (
	"github.com/mutalisk999/gitbitex-service-group/models"
	"github.com/shopspring/decimal"
	"time"
)

type LogType string

const (
	LogTypeMatch = LogType("match")
	LogTypeOpen  = LogType("open")
	LogTypeDone  = LogType("done")
)

type Log interface {
	GetSeq() int64
}

type Base struct {
	Type      LogType
	Sequence  int64
	ProductId string
	Time      time.Time
}

type ReceivedLog struct {
	Base
	OrderId   int64
	Size      decimal.Decimal
	Price     decimal.Decimal
	Side      models.Side
	OrderType models.OrderType
}

func (l *ReceivedLog) GetSeq() int64 {
	return l.Sequence
}

type OpenLog struct {
	Base
	OrderId       int64
	UserId        int64
	RemainingSize decimal.Decimal
	Price         decimal.Decimal
	Side          models.Side
	TimeInForce   models.TimeInForceType
}

func newOpenLog(logSeq int64, productId string, takerOrder *BookOrder) *OpenLog {
	return &OpenLog{
		Base:          Base{LogTypeOpen, logSeq, productId, time.Now()},
		OrderId:       takerOrder.OrderId,
		UserId:        takerOrder.UserId,
		RemainingSize: takerOrder.Size,
		Price:         takerOrder.Price,
		Side:          takerOrder.Side,
		TimeInForce:   takerOrder.TimeInForce,
	}
}

func (l *OpenLog) GetSeq() int64 {
	return l.Sequence
}

type DoneLog struct {
	Base
	OrderId       int64
	UserId        int64
	Price         decimal.Decimal
	RemainingSize decimal.Decimal
	Reason        models.DoneReason
	Side          models.Side
	TimeInForce   models.TimeInForceType
}

func newDoneLog(logSeq int64, productId string, order *BookOrder, remainingSize decimal.Decimal, reason models.DoneReason) *DoneLog {
	return &DoneLog{
		Base:          Base{LogTypeDone, logSeq, productId, time.Now()},
		OrderId:       order.OrderId,
		UserId:        order.UserId,
		Price:         order.Price,
		RemainingSize: remainingSize,
		Reason:        reason,
		Side:          order.Side,
		TimeInForce:   order.TimeInForce,
	}
}

func (l *DoneLog) GetSeq() int64 {
	return l.Sequence
}

type MatchLog struct {
	Base
	TradeSeq         int64
	TakerOrderId     int64
	MakerOrderId     int64
	TakerUserId      int64
	MakerUserId      int64
	Side             models.Side
	Price            decimal.Decimal
	Size             decimal.Decimal
	TakerTimeInForce models.TimeInForceType
	MakerTimeInForce models.TimeInForceType
}

func newMatchLog(logSeq int64, productId string, tradeSeq int64, takerOrder, makerOrder *BookOrder, price, size decimal.Decimal) *MatchLog {
	return &MatchLog{
		Base:             Base{LogTypeMatch, logSeq, productId, time.Now()},
		TradeSeq:         tradeSeq,
		TakerOrderId:     takerOrder.OrderId,
		MakerOrderId:     makerOrder.OrderId,
		TakerUserId:      takerOrder.UserId,
		MakerUserId:      makerOrder.UserId,
		Side:             makerOrder.Side,
		Price:            price,
		Size:             size,
		TakerTimeInForce: takerOrder.TimeInForce,
		MakerTimeInForce: makerOrder.TimeInForce,
	}
}

func (l *MatchLog) GetSeq() int64 {
	return l.Sequence
}
