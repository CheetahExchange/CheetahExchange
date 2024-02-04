package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/matching"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/CheetahExchange/CheetahExchange/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var productId2Writer sync.Map

func getWriter(productId string) *kafka.Writer {
	writer, found := productId2Writer.Load(productId)
	if found {
		return writer.(*kafka.Writer)
	}

	gbeConfig := conf.GetConfig()

	newWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      gbeConfig.Kafka.Brokers,
		Topic:        matching.TopicOrderPrefix + productId,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 5 * time.Millisecond,
	})
	productId2Writer.Store(productId, newWriter)
	return newWriter
}

func submitOrder(order *models.Order) {
	buf, err := json.Marshal(order)
	if err != nil {
		log.Error(err)
		return
	}

	err = getWriter(order.ProductId).WriteMessages(context.Background(), kafka.Message{Value: buf})
	if err != nil {
		log.Error(err)
	}
}

// POST /orders
func PlaceOrder(ctx *gin.Context) {
	var req placeOrderRequest
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	side := models.Side(req.Side)
	if len(side) == 0 {
		side = models.SideBuy
	}

	orderType := models.OrderType(req.Type)
	if len(orderType) == 0 {
		orderType = models.OrderTypeLimit
	}

	timeInForce := models.TimeInForceType(req.TimeInForce)
	if len(timeInForce) == 0 {
		timeInForce = models.GoodTillCanceled
	}

	if len(req.ClientOid) > 0 {
		_, err = uuid.Parse(req.ClientOid)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newMessageVo(fmt.Errorf("invalid client_oid: %v", err)))
			return
		}
	}

	//todo
	//size, err := utils.StringToFloat64(req.size)
	//price, err := utils.StringToFloat64(req.price)
	size := decimal.NewFromFloat(req.Size)
	price := decimal.NewFromFloat(req.Price)
	funds := decimal.NewFromFloat(req.Funds)

	order, err := service.PlaceOrder(GetCurrentUser(ctx).Id, GetCurrentUser(ctx).UserLevel, req.ClientOid,
		req.ProductId, orderType, timeInForce,
		side, size, price, funds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	submitOrder(order)

	ctx.JSON(http.StatusOK, order)
}

// 撤销指定id的订单
// DELETE /orders/1
// DELETE /orders/client:1
func CancelOrder(ctx *gin.Context) {
	rawOrderId := ctx.Param("orderId")

	var order *models.Order
	var err error
	if strings.HasPrefix(rawOrderId, "client:") {
		clientOid := strings.Split(rawOrderId, ":")[1]
		order, err = service.GetOrderByClientOid(GetCurrentUser(ctx).Id, clientOid)
	} else {
		orderId, _ := utils.AToInt64(rawOrderId)
		order, err = service.GetOrderById(orderId)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	if order == nil || order.UserId != GetCurrentUser(ctx).Id {
		ctx.JSON(http.StatusNotFound, newMessageVo(errors.New("order not found")))
		return
	}

	order.Status = models.OrderStatusCancelling
	submitOrder(order)

	ctx.JSON(http.StatusOK, nil)
}

// 批量撤单
// DELETE /orders/?productId=BTC-USDT&side=[buy,sell]
func CancelOrders(ctx *gin.Context) {
	productId := ctx.Query("productId")

	var side *models.Side
	var err error
	rawSide := ctx.Query("side")
	if len(rawSide) > 0 {
		side, err = models.NewSideFromString(rawSide)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newMessageVo(err))
			return
		}
	}

	orders, err := service.GetOrdersByUserId(GetCurrentUser(ctx).Id,
		[]models.OrderStatus{models.OrderStatusOpen, models.OrderStatusNew}, side, productId, 0, 0, 10000)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	for _, order := range orders {
		order.Status = models.OrderStatusCancelling
		submitOrder(order)
	}

	ctx.JSON(http.StatusOK, nil)
}

// GET /orders
func GetOrders(ctx *gin.Context) {
	productId := ctx.Query("productId")

	var side *models.Side
	var err error
	rawSide := ctx.GetString("side")
	if len(rawSide) > 0 {
		side, err = models.NewSideFromString(rawSide)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newMessageVo(err))
			return
		}
	}

	var statuses []models.OrderStatus
	statusValues := ctx.QueryArray("status")
	for _, statusValue := range statusValues {
		status, err := models.NewOrderStatusFromString(statusValue)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newMessageVo(err))
			return
		}
		statuses = append(statuses, *status)
	}

	before, _ := strconv.ParseInt(ctx.Query("before"), 10, 64)
	after, _ := strconv.ParseInt(ctx.Query("after"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query("limit"), 10, 64)

	orders, err := service.GetOrdersByUserId(GetCurrentUser(ctx).Id, statuses, side, productId, before, after, int(limit))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	orderVos := []*orderVo{}
	for _, order := range orders {
		orderVos = append(orderVos, newOrderVo(order))
	}

	var newBefore, newAfter int64 = 0, 0
	if len(orders) > 0 {
		newBefore = orders[0].Id
		newAfter = orders[len(orders)-1].Id
	}
	ctx.Header("gbe-before", strconv.FormatInt(newBefore, 10))
	ctx.Header("gbe-after", strconv.FormatInt(newAfter, 10))

	ctx.JSON(http.StatusOK, orderVos)
}
