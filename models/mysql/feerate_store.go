package mysql

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
)

func (s *Store) GetFeeRateByUserLevel(userLevel string) (*models.FeeRate, error) {
	var feeRate models.FeeRate
	err := s.db.Where("user_level =?", userLevel).Limit(1).Find(&feeRate).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &feeRate, err
}
