package mysql

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/jinzhu/gorm"
)

func (s *Store) GetTicksByProductId(productId string, granularity int64, beforeTime, afterTime int64, limit int) ([]*models.Tick, error) {
	var ticks []*models.Tick
	db := s.db.Table("g_tick").Where("product_id =?", productId).Where("granularity =?", granularity)

	if beforeTime > 0 {
		db = db.Where("time >?", beforeTime)
	}

	if afterTime > 0 {
		db = db.Where("time <?", afterTime)
	}

	if limit <= 0 {
		limit = 100
	}

	db = db.Order("time DESC").Limit(limit)
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
	sql := "INSERT INTO g_tick (created_at,product_id,granularity,time,open,low,high,close," +
		"volume,quote_volume,log_offset,log_seq) VALUES (NOW(),?,?,?,?,?,?,?,?,?,?,?) " +
		"ON DUPLICATE KEY UPDATE created_at = NOW(),open=VALUES(open),low=VALUES(low),high=VALUES(high),close=VALUES(close)," +
		"volume=VALUES(volume),quote_volume=VALUES(quote_volume),log_offset=VALUES(log_offset),log_seq=VALUES(log_seq)"
	for _, tick := range ticks {
		if tick == nil {
			continue
		}
		args := []interface{}{
			tick.ProductId, tick.Granularity, tick.Time, tick.Open, tick.Low, tick.High, tick.Close,
			tick.Volume, tick.QuoteVolume, tick.LogOffset, tick.LogSeq,
		}
		if err := s.db.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddOrUpdateTick(tick *models.Tick) error {
	if tick == nil {
		return nil
	}
	sql := "INSERT INTO g_tick (created_at,product_id,granularity,time,open,low,high,close," +
		"volume,quote_volume,log_offset,log_seq) VALUES (NOW(),?,?,?,?,?,?,?,?,?,?,?) " +
		"ON DUPLICATE KEY UPDATE created_at = NOW(),open=?,low=?,high=?,close=?," +
		"volume=?,quote_volume=?,log_offset=?,log_seq=?"
	args := []interface{}{
		tick.ProductId, tick.Granularity, tick.Time, tick.Open, tick.Low, tick.High, tick.Close,
		tick.Volume, tick.QuoteVolume, tick.LogOffset, tick.LogSeq,
		tick.Open, tick.Low, tick.High, tick.Close,
		tick.Volume, tick.QuoteVolume, tick.LogOffset, tick.LogSeq,
	}
	return s.db.Exec(sql, args...).Error
}
