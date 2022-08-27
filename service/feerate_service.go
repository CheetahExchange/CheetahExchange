package service

import (
	"github.com/mutalisk999/gitbitex-service-group/models"
	"github.com/mutalisk999/gitbitex-service-group/models/mysql"
)

func GetFeeRateByUserLevel(userLevel string) (*models.FeeRate, error) {
	return mysql.SharedStore().GetFeeRateByUserLevel(userLevel)
}
