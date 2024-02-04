package service

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
)

func GetFeeRateByUserLevel(userLevel string) (*models.FeeRate, error) {
	return mysql.SharedStore().GetFeeRateByUserLevel(userLevel)
}
