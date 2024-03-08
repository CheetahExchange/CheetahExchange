package mysql

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
	"time"
)

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := s.db.Table("g_user").Where("email =?", email).Scan(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (s *Store) AddUser(user *models.User) error {
	user.CreatedAt = time.Now()
	return s.db.Create(user).Error
}

func (s *Store) UpdateUser(user *models.User) error {
	return s.db.Save(user).Error

}
