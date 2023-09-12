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

package pushing

import "fmt"

type Level2Type string
type Channel string

func (t Channel) Format(productId string, userId int64) string {
	return fmt.Sprintf("%v:%v:%v", t, productId, userId)
}

func (t Channel) FormatWithUserId(userId int64) string {
	return fmt.Sprintf("%v:%v", t, userId)
}

func (t Channel) FormatWithProductId(productId string) string {
	return fmt.Sprintf("%v:%v", t, productId)
}

func CandlesFormatWithGranularityProductId(granularity int64, productId string) string {
	return fmt.Sprintf("candles_%dm:%v", granularity, productId)
}

const (
	Level2TypeSnapshot = Level2Type("snapshot")
	Level2TypeUpdate   = Level2Type("l2update")

	ChannelTicker = Channel("ticker")
	ChannelMatch  = Channel("match")
	ChannelTrade  = Channel("trade")
	ChannelLevel2 = Channel("level2")
	ChannelFunds  = Channel("funds")
	ChannelOrder  = Channel("order")

	ChannelCandles1m    = Channel("candles_1m")
	ChannelCandles3m    = Channel("candles_3m")
	ChannelCandles5m    = Channel("candles_5m")
	ChannelCandles15m   = Channel("candles_15m")
	ChannelCandles30m   = Channel("candles_30m")
	ChannelCandles60m   = Channel("candles_60m")
	ChannelCandles120m  = Channel("candles_120m")
	ChannelCandles240m  = Channel("candles_240m")
	ChannelCandles360m  = Channel("candles_360m")
	ChannelCandles720m  = Channel("candles_720m")
	ChannelCandles1440m = Channel("candles_1440m")
)

type Request struct {
	Type        string   `json:"type"`
	ProductIds  []string `json:"product_ids"`
	CurrencyIds []string `json:"currency_ids"`
	Channels    []string `json:"channels"`
	Token       string   `json:"token"`
}

type Response struct {
	Type       string   `json:"type"`
	ProductIds []string `json:"product_ids"`
	Channels   []string `json:"channels"`
	Token      string   `json:"token"`
}

type Level2SnapshotMessage struct {
	Type      Level2Type       `json:"type"`
	ProductId string           `json:"productId"`
	Bids      [][3]interface{} `json:"bids"` // [["6500.15", "0.57753524"]]
	Asks      [][3]interface{} `json:"asks"`
}

type Level2UpdateMessage struct {
	Type      Level2Type       `json:"type"`
	ProductId string           `json:"productId"`
	Changes   [][3]interface{} `json:"changes"` // ["buy", "6500.09", "0.84702376"],
}

type Level2Change struct {
	Seq       int64
	ProductId string
	Side      string
	Price     string
	Size      string
}

type MatchMessage struct {
	Type         string `json:"type"`
	TradeSeq     int64  `json:"tradeSeq"`
	Sequence     int64  `json:"sequence"`
	Time         string `json:"time"`
	ProductId    string `json:"productId"`
	Price        string `json:"price"`
	Size         string `json:"size"`
	MakerOrderId string `json:"makerOrderId"`
	TakerOrderId string `json:"takerOrderId"`
	Side         string `json:"side"`
}

type TickerMessage struct {
	Type      string `json:"type"`
	TradeSeq  int64  `json:"tradeSeq"`
	Sequence  int64  `json:"sequence"`
	Time      string `json:"time"`
	ProductId string `json:"productId"`
	Price     string `json:"price"`
	Side      string `json:"side"`
	LastSize  string `json:"lastSize"`
	BestBid   string `json:"bestBid"`
	BestAsk   string `json:"bestAsk"`
	Volume24h string `json:"volume24h"`
	Volume30d string `json:"volume30d"`
	Low24h    string `json:"low24h"`
	Open24h   string `json:"open24h"`
}

type CandlesMessage struct {
	Type      string `json:"type"`
	ProductId string `json:"productId"`
	Time      string `json:"time"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	Low       string `json:"low"`
	High      string `json:"high"`
	Volume    string `json:"volume"`
}

type FundsMessage struct {
	Type      string `json:"type"`
	Sequence  int64  `json:"sequence"`
	UserId    string `json:"userId"`
	Currency  string `json:"currencyCode"`
	Available string `json:"available"`
	Hold      string `json:"hold"`
}

type OrderMessage struct {
	UserId        int64  `json:"userId"`
	Type          string `json:"type"`
	Sequence      int64  `json:"sequence"`
	Id            string `json:"id"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	Funds         string `json:"funds"`
	ProductId     string `json:"productId"`
	Side          string `json:"side"`
	OrderType     string `json:"orderType"`
	CreatedAt     string `json:"createdAt"`
	FillFees      string `json:"fillFees"`
	FilledSize    string `json:"filledSize"`
	ExecutedValue string `json:"executedValue"`
	Status        string `json:"status"`
	Settled       bool   `json:"settled"`
}

type TradeMessage struct {
	Type         string `json:"type"`
	Time         string `json:"time"`
	ProductId    string `json:"productId"`
	Price        string `json:"price"`
	Size         string `json:"size"`
	MakerOrderId string `json:"makerOrderId"`
	TakerOrderId string `json:"takerOrderId"`
	Side         string `json:"side"`
}
