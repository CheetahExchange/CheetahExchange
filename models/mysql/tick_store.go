package mysql

import (
	"fmt"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
	"strings"
	"time"
)

func (s *Store) GetTicksByProductId(productId string, granularity int64, limit int) ([]*models.Tick, error) {
	var ticks []*models.Tick
	db := s.db.Table("g_tick").Where("product_id =?", productId).Where("granularity =?", granularity).
		Order("time DESC").Limit(limit)
	err := db.Find(&ticks).Error
	return ticks, err
}

func (s *Store) GetLastTickByProductId(productId string, granularity int64) (*models.Tick, error) {
	var tick models.Tick
	err := s.db.Table("g_tick").Where("product_id =?", productId).Where("granularity =?", granularity).
		Order("time DESC").Limit(1).Scan(&tick).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &tick, err
}

func (s *Store) GetLastTicksAllByProductId(productId string) ([]*models.Tick, error) {
	var ticks []*models.Tick
	err := s.db.Raw("SELECT t1.* FROM "+
		" (SELECT * FROM g_tick WHERE product_id =?) as t1, "+
		" (SELECT max(time) as last, granularity FROM g_tick WHERE product_id =? GROUP BY granularity) as t2 "+
		" WHERE t1.time = t2.last AND t1.granularity = t2.granularity ",
		productId, productId).Scan(&ticks).Error
	return ticks, err
}

func (s *Store) AddTicks(ticks []*models.Tick) error {
	if len(ticks) == 0 {
		return nil
	}
	var valueStrings []string
	for _, tick := range ticks {
		valueString := fmt.Sprintf("'%v','%v',%v,%v,%v,%v,%v,%v,%v,%v,%v)",
			time.Now(), tick.ProductId, tick.Granularity, tick.Time, tick.Open, tick.Low, tick.High, tick.Close,
			tick.Volume, tick.LogOffset, tick.LogSeq)
		valueStrings = append(valueStrings, valueString)
	}
	sql := fmt.Sprintf("REPLACE INTO g_tick(created_at, product_id,granularity,time,open,low,high,close,"+
		"volume,log_offset,log_seq) VALUES %s", strings.Join(valueStrings, ","))
	return s.db.Exec(sql).Error
}
