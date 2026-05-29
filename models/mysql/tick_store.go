package mysql

import (
	"strings"

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
	baseSql := "INSERT INTO g_tick (created_at,product_id,granularity,time,open,low,high,close," +
		"volume,quote_volume,log_offset,log_seq) VALUES "
	onDuplicate := " AS new ON DUPLICATE KEY UPDATE created_at = NOW(),open=new.open,low=new.low," +
		"high=new.high,close=new.close,volume=new.volume," +
		"quote_volume=new.quote_volume,log_offset=new.log_offset,log_seq=new.log_seq"

	// batch in chunks of 500 to stay within MySQL limits
	for i := 0; i < len(ticks); i += 500 {
		end := i + 500
		if end > len(ticks) {
			end = len(ticks)
		}
		batch := ticks[i:end]

		var placeholders []string
		var args []interface{}
		for _, tick := range batch {
			if tick == nil {
				continue
			}
			placeholders = append(placeholders, "(NOW(),?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, tick.ProductId, tick.Granularity, tick.Time, tick.Open, tick.Low, tick.High, tick.Close,
				tick.Volume, tick.QuoteVolume, tick.LogOffset, tick.LogSeq)
		}
		if len(placeholders) == 0 {
			continue
		}
		if err := s.db.Exec(baseSql+strings.Join(placeholders, ",")+onDuplicate, args...).Error; err != nil {
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
		"volume,quote_volume,log_offset,log_seq) VALUES (NOW(),?,?,?,?,?,?,?,?,?,?,?) AS new " +
		"ON DUPLICATE KEY UPDATE created_at = NOW(),open=new.open,low=new.low,high=new.high,close=new.close," +
		"volume=new.volume,quote_volume=new.quote_volume,log_offset=new.log_offset,log_seq=new.log_seq"
	args := []interface{}{
		tick.ProductId, tick.Granularity, tick.Time, tick.Open, tick.Low, tick.High, tick.Close,
		tick.Volume, tick.QuoteVolume, tick.LogOffset, tick.LogSeq,
	}
	return s.db.Exec(sql, args...).Error
}
