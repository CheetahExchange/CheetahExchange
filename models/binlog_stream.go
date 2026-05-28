package models

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/utils"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
)

type BinLogStream struct {
	canal.DummyEventHandler
	redisClient *redis.Client
}

func IntegerInterfaceToInt64(i interface{}) int64 {
	switch v := i.(type) {
	case int:
		return int64(v)
	case uint:
		return int64(v)
	case int64:
		return v
	case uint64:
		return int64(v)
	case int32:
		return int64(v)
	case uint32:
		return int64(v)
	case int16:
		return int64(v)
	case uint16:
		return int64(v)
	case int8:
		return int64(v)
	case uint8:
		return int64(v)
	default:
		panic(fmt.Sprintf("IntegerInterfaceToInt64: %v", reflect.TypeOf(i)))
	}
}

func IntegerInterfaceToUint64(i interface{}) uint64 {
	switch v := i.(type) {
	case int:
		return uint64(v)
	case uint:
		return uint64(v)
	case int64:
		return uint64(v)
	case uint64:
		return v
	case int32:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int16:
		return uint64(v)
	case uint16:
		return uint64(v)
	case int8:
		return uint64(v)
	case uint8:
		return uint64(v)
	default:
		panic(fmt.Sprintf("IntegerInterfaceToUint64: %v", reflect.TypeOf(i)))
	}
}

func NewBinLogStream() *BinLogStream {
	spotConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     spotConfig.Redis.Addr,
		Password: spotConfig.Redis.Password,
		DB:       0,
	})

	return &BinLogStream{
		redisClient: redisClient,
	}
}

func (s *BinLogStream) OnRow(e *canal.RowsEvent) error {
	switch {
	case strings.HasPrefix(e.Table.Name, "g_order"):
		if e.Action == "delete" {
			return nil
		}

		var n = 0
		if e.Action == "update" {
			n = 1
		}

		var v Order
		s.parseRow(e, e.Rows[n], &v)

		buf, _ := json.Marshal(v)
		ret := s.redisClient.Publish(context.Background(), TopicOrder, buf)
		if ret.Err() != nil {
			log.Error(ret.Err())
		}

	case e.Table.Name == "g_account":
		var n = 0
		if e.Action == "update" {
			n = 1
		}

		var v Account
		s.parseRow(e, e.Rows[n], &v)

		buf, _ := json.Marshal(v)
		ret := s.redisClient.Publish(context.Background(), TopicAccount, buf)
		if ret.Err() != nil {
			log.Error(ret.Err())
		}

	case e.Table.Name == "g_trade":
		if e.Action == "delete" {
			return nil
		}

		var n = 0
		if e.Action == "update" {
			n = 1
		}

		var v Trade
		s.parseRow(e, e.Rows[n], &v)

		buf, _ := json.Marshal(v)
		ret := s.redisClient.Publish(context.Background(), TopicTrade, buf)
		if ret.Err() != nil {
			log.Error(ret.Err())
		}

	case e.Table.Name == "g_fill":
		if e.Action == "delete" || e.Action == "update" {
			return nil
		}

		var v Fill
		s.parseRow(e, e.Rows[0], &v)

		buf, _ := json.Marshal(v)
		ret := s.redisClient.LPush(context.Background(), TopicFill, buf)
		if ret.Err() != nil {
			log.Error(ret.Err())
		}

	case e.Table.Name == "g_bill":
		if e.Action == "delete" || e.Action == "update" {
			return nil
		}

		var v Bill
		s.parseRow(e, e.Rows[0], &v)

		buf, _ := json.Marshal(v)
		ret := s.redisClient.LPush(context.Background(), TopicBill, buf)
		if ret.Err() != nil {
			log.Error(ret.Err())
		}

	}

	return nil
}

func (s *BinLogStream) parseRow(e *canal.RowsEvent, row []interface{}, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)

		colIdx := s.getColumnIndexByName(e, utils.SnakeCase(t.Field(i).Name))
		rowVal := row[colIdx]

		switch f.Type().Name() {
		case "int64", "int32", "int16", "int8", "int":
			f.SetInt(IntegerInterfaceToInt64(rowVal))
		case "uint64", "uint32", "uint16", "uint8", "uint":
			f.SetUint(IntegerInterfaceToUint64(rowVal))
		case "string":
			f.SetString(rowVal.(string))
		case "bool":
			v := IntegerInterfaceToInt64(rowVal)
			if v == 0 {
				f.SetBool(false)
			} else {
				f.SetBool(true)
			}
		case "Time":
			if rowVal != nil {
				f.Set(reflect.ValueOf(rowVal.(time.Time)))
			}
		case "Decimal":
			d, _ := decimal.NewFromString(rowVal.(string))
			f.Set(reflect.ValueOf(d))
		default:
			f.SetString(rowVal.(string))
		}
	}
}

func (s *BinLogStream) getColumnIndexByName(e *canal.RowsEvent, name string) int {
	for id, value := range e.Table.Columns {
		if value.Name == name {
			return id
		}
	}
	return -1
}

func (s *BinLogStream) Start() {
	spotConfig := conf.GetConfig()

	cfg := canal.NewDefaultConfig()
	cfg.Addr = spotConfig.DataSource.Addr
	cfg.User = spotConfig.DataSource.User
	cfg.Password = spotConfig.DataSource.Password
	cfg.Dump.ExecutionPath = ""
	cfg.Dump.TableDB = spotConfig.DataSource.Database
	cfg.ParseTime = true
	cfg.IncludeTableRegex = []string{spotConfig.DataSource.Database + "\\..*"}
	cfg.ExcludeTableRegex = []string{"mysql\\..*"}
	c, err := canal.NewCanal(cfg)
	if err != nil {
		panic(err)
	}
	c.SetEventHandler(s)

	pos, err := c.GetMasterPos()
	if err != nil {
		panic(err)
	}
	err = c.RunFrom(pos)
	if err != nil {
		panic(err)
	}
}
