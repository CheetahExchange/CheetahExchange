package service

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
)

func GetConfigs() ([]*models.Config, error) {
	return mysql.SharedStore().GetConfigs()
}
