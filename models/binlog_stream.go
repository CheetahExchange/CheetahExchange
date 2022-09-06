// Copyright 2019 GitBitEx.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/mutalisk999/gitbitex-service-group/conf"
	"github.com/mutalisk999/gitbitex-service-group/utils"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql/canal"
	"reflect"
	"time"
)

type BinLogStream struct {
	canal.DummyEventHandler
	redisClient *redis.Client
}

func IntegerInterfaceToInt64(i interface{}) int64 {
	switch i.(type) {
	case int:
		return int64(i.(int))
	case uint:
		return int64(i.(uint))
	case int64:
		return i.(int64)
	case uint64:
		return int64(i.(uint64))
	case int32:
		return int64(i.(int32))
	case uint32:
		return int64(i.(uint32))
	case int16:
		return int64(i.(int16))
	case uint16:
		return int64(i.(uint16))
	case int8:
		return int64(i.(int8))
	case uint8:
		return int64(i.(uint8))
	default:
		panic(fmt.Sprintf("IntegerInterfaceToInt64: %v", reflect.TypeOf(i)))
	}
}

func IntegerInterfaceToUint64(i interface{}) uint64 {
	switch i.(type) {
	case int:
		return uint64(i.(int))
	case uint:
		return uint64(i.(uint))
	case int64:
		return uint64(i.(int64))
	case uint64:
		return i.(uint64)
	case int32:
		return uint64(i.(int32))
	case uint32:
		return uint64(i.(uint32))
	case int16:
		return uint64(i.(int16))
	case uint16:
		return uint64(i.(uint16))
	case int8:
		return uint64(i.(int8))
	case uint8:
		return uint64(i.(uint8))
	default:
		panic(fmt.Sprintf("IntegerInterfaceToUint64: %v", reflect.TypeOf(i)))
	}
}

func NewBinLogStream() *BinLogStream {
	gbeConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     gbeConfig.Redis.Addr,
		Password: gbeConfig.Redis.Password,
		DB:       0,
	})

	return &BinLogStream{
		redisClient: redisClient,
	}
}

func (s *BinLogStream) OnRow(e *canal.RowsEvent) error {
	switch e.Table.Name {
	case "g_order":
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

	case "g_account":
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

	case "g_trade":
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

	case "g_fill":
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

	case "g_bill":
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
			d := decimal.NewFromFloat(rowVal.(float64))
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
	gbeConfig := conf.GetConfig()

	cfg := canal.NewDefaultConfig()
	cfg.Addr = gbeConfig.DataSource.Addr
	cfg.User = gbeConfig.DataSource.User
	cfg.Password = gbeConfig.DataSource.Password
	cfg.Dump.ExecutionPath = ""
	cfg.Dump.TableDB = gbeConfig.DataSource.Database
	cfg.ParseTime = true
	cfg.IncludeTableRegex = []string{gbeConfig.DataSource.Database + "\\..*"}
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
