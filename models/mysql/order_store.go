package mysql

import (
	"fmt"
	"time"

	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
)

func GetTableIndexByUserId(userId int64) int {
	return int(userId % models.TableOrderSplitCount)
}

func GetTableIndexByOrderId(orderId int64) int {
	// return int((orderId >> 12) % (1 << 10))
	return int((orderId >> 12) % int64(models.TableOrderSplitCount))
}

func (s *Store) GetOrderById(orderId int64) (*models.Order, error) {
	var order models.Order
	table := fmt.Sprintf("g_order_%d", GetTableIndexByOrderId(orderId))
	err := s.db.Table(table).Where("id =?", orderId).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrderByClientOid(userId int64, clientOid string) (*models.Order, error) {
	var order models.Order
	table := fmt.Sprintf("g_order_%d", GetTableIndexByUserId(userId))
	err := s.db.Table(table).Where("user_id =?", userId).Where("client_oid =?", clientOid).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrderByIdForUpdate(orderId int64) (*models.Order, error) {
	var order models.Order
	table := fmt.Sprintf("g_order_%d", GetTableIndexByOrderId(orderId))
	err := s.db.Table(table).Raw("SELECT * FROM ? WHERE id =? FOR UPDATE", table, orderId).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrdersByUserId(userId int64, statuses []models.OrderStatus, side *models.Side, productId string,
	beforeId, afterId int64, limit int) ([]*models.Order, error) {
	table := fmt.Sprintf("g_order_%d", GetTableIndexByUserId(userId))
	db := s.db.Table(table).Where("user_id =?", userId)

	if len(statuses) != 0 {
		db = db.Where("status IN (?)", statuses)
	}

	if len(productId) != 0 {
		db = db.Where("product_id =?", productId)
	}

	if side != nil {
		db = db.Where("side =?", side)
	}

	if beforeId > 0 {
		db = db.Where("id >?", beforeId)
	}

	if afterId > 0 {
		db = db.Where("id <?", afterId)
	}

	if limit <= 0 {
		limit = 100
	}

	db = db.Order("id DESC").Limit(limit)

	var orders []*models.Order
	err := db.Find(&orders).Error
	return orders, err
}

func (s *Store) AddOrder(order *models.Order) error {
	order.CreatedAt = time.Now()
	table := fmt.Sprintf("g_order_%d", GetTableIndexByUserId(order.UserId))
	return s.db.Table(table).Create(order).Error
}

func (s *Store) UpdateOrder(order *models.Order) error {
	order.UpdatedAt = time.Now()
	table := fmt.Sprintf("g_order_%d", GetTableIndexByUserId(order.UserId))
	return s.db.Table(table).Save(order).Error
}

func (s *Store) UpdateOrderStatus(orderId int64, oldStatus, newStatus models.OrderStatus) (bool, error) {
	table := fmt.Sprintf("g_order_%d", GetTableIndexByOrderId(orderId))
	ret := s.db.Table(table).Where("id =? AND status =?", orderId, oldStatus).
		Updates(models.Order{Status: newStatus, UpdatedAt: time.Now()})
	if ret.Error != nil {
		return false, ret.Error
	}
	return ret.RowsAffected > 0, nil
}
