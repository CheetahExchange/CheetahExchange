package matching

import (
	"fmt"
	"math"

	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
)

const (
	orderIdWindowCap = 10000
)

type orderBook struct {
	// one product corresponds to one order book
	product *models.Product

	// depths: asks & bids
	depths map[models.Side]*depth

	// strictly continuously increasing transaction ID, used for the primary key ID of trade
	tradeSeq int64

	// strictly continuously increasing log SEQ, used to write matching log
	logSeq int64

	// to prevent the order from being submitted to the order book repeatedly,
	// a sliding window de duplication strategy is adopted.
	orderIdWindows []*Window
}

type orderBookSnapshot struct {
	// order book product id
	ProductId string

	// all orders
	Orders []BookOrder

	// trade seq at snapshot time
	TradeSeq int64

	// log seq at snapshot time
	LogSeq int64

	// state of de duplication window
	OrderIdWindows []*Window
}

type priceOrderIdKey struct {
	price   decimal.Decimal
	orderId uint64
}

func NewOrderBook(product *models.Product) *orderBook {
	asks := &depth{
		queue:  treemap.NewWith(priceOrderIdKeyAscComparator),
		orders: map[uint64]*BookOrder{},
	}
	bids := &depth{
		queue:  treemap.NewWith(priceOrderIdKeyDescComparator),
		orders: map[uint64]*BookOrder{},
	}

	orderBook := &orderBook{
		product:        product,
		depths:         map[models.Side]*depth{models.SideBuy: bids, models.SideSell: asks},
		orderIdWindows: make([]*Window, 0),
	}
	for i := 0; i < models.TableOrderSplitCount; i++ {
		orderBook.orderIdWindows = append(orderBook.orderIdWindows, newWindow(0, orderIdWindowCap))
	}
	return orderBook
}

// prepareTakerOrder adjusts market order price for matching:
// Market-Buy → MaxFloat32, Market-Sell → Zero, ensuring prices will cross.
func prepareTakerOrder(takerOrder *BookOrder) {
	if takerOrder.Type == models.OrderTypeMarket {
		if takerOrder.Side == models.SideBuy {
			takerOrder.Price = decimal.NewFromFloat(math.MaxFloat32)
		} else {
			takerOrder.Price = decimal.Zero
		}
	}
}

// pricesCross checks whether taker and maker prices cross.
func pricesCross(takerSide models.Side, takerPrice, makerPrice decimal.Decimal) bool {
	if takerSide == models.SideBuy {
		return takerPrice.GreaterThanOrEqual(makerPrice)
	}
	return takerPrice.LessThanOrEqual(makerPrice)
}

// calcTradeSize calculates the trade size for a single taker-maker pair,
// adjusting taker's remaining Size or Funds accordingly.
// Returns (size, fundsConsumed) and whether taker is exhausted.
func calcTradeSize(takerOrder *BookOrder, makerOrder *BookOrder, product *models.Product) (size decimal.Decimal, fundsConsumed decimal.Decimal, takerExhausted bool) {
	if takerOrder.Type == models.OrderTypeLimit ||
		(takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideSell) {
		if takerOrder.Size.IsZero() {
			return decimal.Zero, decimal.Zero, true
		}
		size = decimal.Min(takerOrder.Size, makerOrder.Size)
		takerOrder.Size = takerOrder.Size.Sub(size)
		takerExhausted = takerOrder.Size.IsZero()
		return size, decimal.Zero, takerExhausted
	}

	if takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideBuy {
		if takerOrder.Funds.IsZero() {
			return decimal.Zero, decimal.Zero, true
		}
		takerSize := takerOrder.Funds.Div(makerOrder.Price).Truncate(product.BaseScale)
		if takerSize.IsZero() {
			return decimal.Zero, decimal.Zero, true
		}
		size = decimal.Min(takerSize, makerOrder.Size)
		fundsConsumed = size.Mul(makerOrder.Price)
		takerOrder.Funds = takerOrder.Funds.Sub(fundsConsumed)
		takerExhausted = takerOrder.Funds.IsZero()
		return size, fundsConsumed, takerExhausted
	}

	log.Fatal("unknown orderType and side combination")
	return decimal.Zero, decimal.Zero, true
}

func (o *orderBook) IsOrderWillNotMatch(order *models.Order) bool {
	takerOrder := newBookOrder(order)
	prepareTakerOrder(takerOrder)

	makerDepth := o.depths[takerOrder.Side.Opposite()]

	if takerOrder.Side == models.SideBuy {
		k, v := makerDepth.queue.Min()
		if k == nil || v == nil {
			return true
		}
		makerOrder := makerDepth.orders[v.(uint64)]
		return !pricesCross(takerOrder.Side, takerOrder.Price, makerOrder.Price)
	}

	k, v := makerDepth.queue.Max()
	if k == nil || v == nil {
		return true
	}
	makerOrder := makerDepth.orders[v.(uint64)]
	return !pricesCross(takerOrder.Side, takerOrder.Price, makerOrder.Price)
}

func (o *orderBook) IsOrderWillFullMatch(order *models.Order) bool {
	takerOrder := newBookOrder(order)
	prepareTakerOrder(takerOrder)

	makerDepth := o.depths[takerOrder.Side.Opposite()]
	for itr := makerDepth.queue.Iterator(); itr.Next(); {
		makerOrder := makerDepth.orders[itr.Value().(uint64)]

		if !pricesCross(takerOrder.Side, takerOrder.Price, makerOrder.Price) {
			break
		}

		size, _, _ := calcTradeSize(takerOrder, makerOrder, o.product)
		if size.IsZero() {
			break
		}

		if takerOrder.Type == models.OrderTypeLimit ||
			(takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideSell) {
			if takerOrder.Size.IsZero() {
				break
			}
		} else if takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideBuy {
			if takerOrder.Funds.IsZero() {
				break
			}
		}
	}

	if takerOrder.Type == models.OrderTypeLimit && takerOrder.Size.GreaterThan(decimal.Zero) {
		return false
	}

	return true
}

func (o *orderBook) ApplyOrder(order *models.Order) (logs []Log) {
	// prevent orders from being submitted repeatedly to the matching engine
	idx := mysql.GetTableIndexByOrderId(order.Id)
	err := o.orderIdWindows[idx].put(order.Id)
	if err != nil {
		log.Error(err)
		return logs
	}

	takerOrder := newBookOrder(order)
	prepareTakerOrder(takerOrder)

	makerDepth := o.depths[takerOrder.Side.Opposite()]
	for itr := makerDepth.queue.Iterator(); itr.Next(); {
		makerOrder := makerDepth.orders[itr.Value().(uint64)]

		if !pricesCross(takerOrder.Side, takerOrder.Price, makerOrder.Price) {
			break
		}

		size, _, _ := calcTradeSize(takerOrder, makerOrder, o.product)
		if size.IsZero() {
			break
		}

		// adjust the size of maker order
		err := makerDepth.decrSize(makerOrder.OrderId, size)
		if err != nil {
			log.Fatal(err)
		}

		// matched, write a log
		matchLog := newMatchLog(o.nextLogSeq(), o.product.Id, o.nextTradeSeq(), takerOrder, makerOrder, makerOrder.Price, size)
		logs = append(logs, matchLog)

		// maker is filled
		if makerOrder.Size.IsZero() {
			doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, makerOrder, makerOrder.Size, models.DoneReasonFilled)
			logs = append(logs, doneLog)
		}

		// check if taker is exhausted after this match — next iteration would see zero Size/Funds
		if takerOrder.Type == models.OrderTypeLimit ||
			(takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideSell) {
			if takerOrder.Size.IsZero() {
				break
			}
		} else if takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideBuy {
			if takerOrder.Funds.IsZero() {
				break
			}
		}
	}

	if takerOrder.Type == models.OrderTypeLimit && takerOrder.Size.GreaterThan(decimal.Zero) {
		// If taker has an uncompleted size, put taker in orderBook
		o.depths[takerOrder.Side].add(*takerOrder)

		openLog := newOpenLog(o.nextLogSeq(), o.product.Id, takerOrder)
		logs = append(logs, openLog)

	} else {
		var remainingSize = takerOrder.Size
		var reason = models.DoneReasonFilled

		if takerOrder.Type == models.OrderTypeMarket {
			takerOrder.Price = decimal.Zero
			remainingSize = decimal.Zero
			if (takerOrder.Side == models.SideSell && takerOrder.Size.GreaterThan(decimal.Zero)) ||
				(takerOrder.Side == models.SideBuy && takerOrder.Funds.GreaterThan(decimal.Zero)) {
				reason = models.DoneReasonCancelled
			}
		}

		doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, takerOrder, remainingSize, reason)
		logs = append(logs, doneLog)
	}
	return logs
}

func (o *orderBook) CancelOrder(order *models.Order) (logs []Log) {
	idx := mysql.GetTableIndexByOrderId(order.Id)
	_ = o.orderIdWindows[idx].put(order.Id)

	bookOrder, found := o.depths[order.Side].orders[order.Id]
	if !found {
		return logs
	}

	// Decrease the entire size of the order, which is equivalent to the remove operation.
	remainingSize := bookOrder.Size
	err := o.depths[order.Side].decrSize(order.Id, bookOrder.Size)
	if err != nil {
		panic(err)
	}

	doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, bookOrder, remainingSize, models.DoneReasonCancelled)
	return append(logs, doneLog)
}

func (o *orderBook) NullifyOrder(order *models.Order) (logs []Log) {
	idx := mysql.GetTableIndexByOrderId(order.Id)
	_ = o.orderIdWindows[idx].put(order.Id)

	bookOrder := newBookOrder(order)
	doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, bookOrder, order.Size, models.DoneReasonCancelled)
	return append(logs, doneLog)
}

func (o *orderBook) Snapshot() orderBookSnapshot {
	snapshot := orderBookSnapshot{
		Orders:         make([]BookOrder, len(o.depths[models.SideSell].orders)+len(o.depths[models.SideBuy].orders)),
		LogSeq:         o.logSeq,
		TradeSeq:       o.tradeSeq,
		OrderIdWindows: o.orderIdWindows,
	}

	i := 0
	for _, order := range o.depths[models.SideSell].orders {
		snapshot.Orders[i] = *order
		i++
	}
	for _, order := range o.depths[models.SideBuy].orders {
		snapshot.Orders[i] = *order
		i++
	}

	return snapshot
}

func (o *orderBook) Restore(snapshot *orderBookSnapshot) {
	o.logSeq = snapshot.LogSeq
	o.tradeSeq = snapshot.TradeSeq
	o.orderIdWindows = snapshot.OrderIdWindows
	// if o.orderIdWindows[0].Cap == 0 {
	// 	o.orderIdWindows[0] = newWindow(0, orderIdWindowCap)
	// }
	if len(o.orderIdWindows) != models.TableOrderSplitCount {
		o.orderIdWindows = make([]*Window, 0)
		for i := 0; i < models.TableOrderSplitCount; i++ {
			o.orderIdWindows = append(o.orderIdWindows, newWindow(0, orderIdWindowCap))
		}
	}

	for _, order := range snapshot.Orders {
		o.depths[order.Side].add(order)
	}
}

func (o *orderBook) nextLogSeq() int64 {
	o.logSeq++
	return o.logSeq
}

func (o *orderBook) nextTradeSeq() int64 {
	o.tradeSeq++
	return o.tradeSeq
}

type depth struct {
	// all orders
	orders map[uint64]*BookOrder

	// price first, time first order queue for order match
	// priceOrderIdKey -> orderId
	queue *treemap.Map
}

func (d *depth) add(order BookOrder) {
	d.orders[order.OrderId] = &order
	d.queue.Put(&priceOrderIdKey{order.Price, order.OrderId}, order.OrderId)
}

func (d *depth) decrSize(orderId uint64, size decimal.Decimal) error {
	order, found := d.orders[orderId]
	if !found {
		return fmt.Errorf("order %v not found on book", orderId)
	}

	if order.Size.LessThan(size) {
		return fmt.Errorf("order %v Size %v less than %v", orderId, order.Size, size)
	}

	order.Size = order.Size.Sub(size)
	if order.Size.IsZero() {
		delete(d.orders, orderId)
		d.queue.Remove(&priceOrderIdKey{order.Price, order.OrderId})
	}

	return nil
}

type BookOrder struct {
	OrderId     uint64
	UserId      uint64
	Size        decimal.Decimal
	Funds       decimal.Decimal
	Price       decimal.Decimal
	Side        models.Side
	Type        models.OrderType
	TimeInForce models.TimeInForceType
}

func newBookOrder(order *models.Order) *BookOrder {
	return &BookOrder{
		OrderId:     order.Id,
		UserId:      order.UserId,
		Size:        order.Size,
		Funds:       order.Funds,
		Price:       order.Price,
		Side:        order.Side,
		Type:        order.Type,
		TimeInForce: order.TimeInForce,
	}
}

func priceOrderIdKeyAscComparator(a, b interface{}) int {
	aAsserted := a.(*priceOrderIdKey)
	bAsserted := b.(*priceOrderIdKey)

	x := aAsserted.price.Cmp(bAsserted.price)
	if x != 0 {
		return x
	}

	if aAsserted.orderId < bAsserted.orderId {
		return -1
	} else if aAsserted.orderId > bAsserted.orderId {
		return 1
	}
	return 0
}

func priceOrderIdKeyDescComparator(a, b interface{}) int {
	aAsserted := a.(*priceOrderIdKey)
	bAsserted := b.(*priceOrderIdKey)

	x := aAsserted.price.Cmp(bAsserted.price)
	if x != 0 {
		return -x
	}

	if aAsserted.orderId < bAsserted.orderId {
		return -1
	} else if aAsserted.orderId > bAsserted.orderId {
		return 1
	}
	return 0
}
