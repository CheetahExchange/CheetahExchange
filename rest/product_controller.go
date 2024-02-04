package rest

import (
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/CheetahExchange/CheetahExchange/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
)

// GET /products
func GetProducts(ctx *gin.Context) {
	products, err := service.GetProducts()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	var productVos []*ProductVo
	for _, product := range products {
		productVos = append(productVos, newProductVo(product))
	}

	ctx.JSON(http.StatusOK, productVos)
}

// GET /products/<product-id>/book?level=[1,2,3]
func GetProductOrderBook(ctx *gin.Context) {
	//todo
}

// GET /products/<product-id>/ticker
func GetProductTicker() {
	//todo
}

// GET /products/<product-id>/trades
func GetProductTrades(ctx *gin.Context) {
	productId := ctx.Param("productId")

	var tradeVos []*tradeVo
	trades, _ := service.GetTradesByProductId(productId, 50)
	for _, trade := range trades {
		tradeVos = append(tradeVos, newTradeVo(trade))
	}

	ctx.JSON(http.StatusOK, tradeVos)
}

// GET /products/<product-id>/candles
func GetProductCandles(ctx *gin.Context) {
	productId := ctx.Param("productId")
	granularity, _ := utils.AToInt64(ctx.Query("granularity"))
	limit, _ := utils.AToInt64(ctx.DefaultQuery("limit", "1000"))
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	//[
	//    [ time, low, high, open, close, volume ],
	//    [ 1415398768, 0.32, 4.2, 0.35, 4.2, 12.3 ],
	//]
	var tickVos [][6]float64
	ticks, err := service.GetTicksByProductId(productId, granularity/60, int(limit))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	if len(ticks) > 0 {
		tickTime := ticks[len(ticks)-1].Time

		ticksMap := make(map[int64]*models.Tick)
		for _, tick := range ticks {
			ticksMap[tick.Time] = tick
		}

		lastTick := ticks[len(ticks)-1]
		amendTicks := make([]*models.Tick, 0)
		for {
			if tickTime > time.Now().Unix() {
				break
			}
			tick, ok := ticksMap[tickTime]
			if ok {
				amendTicks = append(amendTicks, tick)
				lastTick = tick
			} else {
				tickAmend := &models.Tick{
					Granularity: granularity,
					Time:        tickTime,
					Open:        lastTick.Close,
					High:        lastTick.Close,
					Low:         lastTick.Close,
					Close:       lastTick.Close,
					Volume:      decimal.NewFromInt(0),
				}
				amendTicks = append(amendTicks, tickAmend)
			}
			tickTime = tickTime + granularity
		}

		amendTicksReversed := make([]*models.Tick, len(amendTicks))
		for i := 0; i < len(amendTicks); i++ {
			amendTicksReversed[i] = amendTicks[len(amendTicks)-i-1]
		}
		if len(amendTicksReversed) > int(limit) {
			amendTicksReversed = amendTicksReversed[0:limit]
		}

		for _, amendTick := range amendTicksReversed {
			tickVos = append(tickVos, [6]float64{float64(amendTick.Time), utils.DToF64(amendTick.Low), utils.DToF64(amendTick.High),
				utils.DToF64(amendTick.Open), utils.DToF64(amendTick.Close), utils.DToF64(amendTick.Volume)})
		}
	}

	ctx.JSON(http.StatusOK, tickVos)
}
