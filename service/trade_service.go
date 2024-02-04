package service

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
)

func GetTradesByProductId(productId string, count int) ([]*models.Trade, error) {
	return mysql.SharedStore().GetTradesByProductId(productId, count)
}

func AddTrades(trades []*models.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	return mysql.SharedStore().AddTrades(trades)
}
