package mysql

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
)

func (s *Store) GetProductById(id string) (*models.Product, error) {
	var product models.Product
	err := s.db.Table("g_product").Where("id =?", id).Scan(&product).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &product, err
}

func (s *Store) GetProducts() ([]*models.Product, error) {
	var products []*models.Product
	err := s.db.Table("g_product").Find(&products).Error
	return products, err
}
