package mysql

import (
	"fmt"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
	"strings"
	"time"
)

func (s *Store) GetLastTradeByProductId(productId string) (*models.Trade, error) {
	var trade models.Trade
	err := s.db.Where("product_id =?", productId).Order("id DESC").Limit(1).Find(&trade).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &trade, err
}

func (s *Store) GetTradesByProductId(productId string, count int) ([]*models.Trade, error) {
	db := s.db.Where("product_id =?", productId).Order("id DESC").Limit(count)
	var trades []*models.Trade
	err := db.Find(&trades).Error
	return trades, err
}

func (s *Store) AddTrades(trades []*models.Trade) error {
	if len(trades) == 0 {
		return nil
	}
	var valueStrings []string
	for _, trade := range trades {
		valueString := fmt.Sprintf("('%v','%v',%v,%v,%v,%v,%v,%v,%v,'%v','%v',%v,%v)",
			time.Now(), trade.ProductId, trade.TradeSeq, trade.TakerOrderId, trade.TakerUserId, trade.MakerOrderId, trade.MakerUserId,
			trade.Price, trade.Size, trade.Side, trade.Time, trade.LogOffset, trade.LogSeq)
		valueStrings = append(valueStrings, valueString)
	}
	sql := fmt.Sprintf("INSERT IGNORE INTO g_trade(created_at,product_id,trade_seq,taker_order_id,taker_user_id,maker_order_id,maker_user_id,"+
		"price,size,side,time,log_offset,log_seq) VALUES %s", strings.Join(valueStrings, ","))
	return s.db.Exec(sql).Error
}
