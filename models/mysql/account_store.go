package mysql

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
	"time"
)

func (s *Store) GetAccount(userId int64, currency string) (*models.Account, error) {
	var account models.Account
	err := s.db.Table("g_account").Where("user_id =?", userId).Where("currency =?", currency).Scan(&account).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &account, err
}

func (s *Store) GetAccountsByUserId(userId int64) ([]*models.Account, error) {
	var accounts []*models.Account
	db := s.db.Table("g_account").Where("user_id =?", userId)
	err := db.Find(&accounts).Error
	return accounts, err
}

func (s *Store) GetAccountForUpdate(userId int64, currency string) (*models.Account, error) {
	var account models.Account
	err := s.db.Raw("SELECT * FROM g_account WHERE user_id =? AND currency =? FOR UPDATE", userId, currency).Scan(&account).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &account, err
}

func (s *Store) AddAccount(account *models.Account) error {
	account.CreatedAt = time.Now()
	return s.db.Create(account).Error
}

func (s *Store) UpdateAccount(account *models.Account) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
