package service

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
)

func GetProductById(id string) (*models.Product, error) {
	return mysql.SharedStore().GetProductById(id)
}

func GetProducts() ([]*models.Product, error) {
	return mysql.SharedStore().GetProducts()
}
