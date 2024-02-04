package service

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
)

func GetUnsettledFills(count int32) ([]*models.Fill, error) {
	return mysql.SharedStore().GetUnsettledFills(count)
}

func AddFills(fills []*models.Fill) error {
	if len(fills) == 0 {
		return nil
	}

	err := mysql.SharedStore().AddFills(fills)
	if err != nil {
		return err
	}
	return nil
}
