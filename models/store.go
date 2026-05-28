package models

type Store interface {
	BeginTx() (Store, error)
	Rollback() error
	CommitTx() error

	GetConfigs() ([]*Config, error)

	GetUserByEmail(email string) (*User, error)
	GetUserById(id uint64) (*User, error)
	AddUser(user *User) error
	UpdateUser(user *User) error

	GetAccount(userId uint64, currency string) (*Account, error)
	GetAccountsByUserId(userId uint64) ([]*Account, error)
	GetAccountForUpdate(userId uint64, currency string) (*Account, error)
	AddAccount(account *Account) error
	UpdateAccount(account *Account) error

	GetUnsettledBillsByUserId(userId uint64, currency string) ([]*Bill, error)
	GetUnsettledBills() ([]*Bill, error)
	AddBills(bills []*Bill) error
	UpdateBill(bill *Bill) error

	GetProductById(id string) (*Product, error)
	GetProducts() ([]*Product, error)

	GetOrderById(orderId uint64) (*Order, error)
	GetOrderByClientOid(userId uint64, clientOid string) (*Order, error)
	GetOrderByIdForUpdate(orderId uint64) (*Order, error)
	GetOrdersByUserId(userId uint64, statuses []OrderStatus, side *Side, productId string,
		beforeId, afterId int64, limit int) ([]*Order, error)
	AddOrder(order *Order) error
	UpdateOrder(order *Order) error
	UpdateOrderStatus(orderId uint64, oldStatus, newStatus OrderStatus) (bool, error)

	GetLastFillByProductId(productId string) (*Fill, error)
	GetUnsettledFillsByOrderId(orderId uint64) ([]*Fill, error)
	GetUnsettledFills(count int32) ([]*Fill, error)
	UpdateFill(fill *Fill) error
	AddFills(fills []*Fill) error

	GetLastTradeByProductId(productId string) (*Trade, error)
	GetTradesByProductId(productId string, count int) ([]*Trade, error)
	AddTrades(trades []*Trade) error

	GetTicksByProductId(productId string, granularity int64, beforeTime, afterTime int64, limit int) ([]*Tick, error)
	GetLastTickByProductId(productId string, granularity int64) (*Tick, error)
	GetLastTicksAllByProductId(productId string) ([]*Tick, error)
	AddTicks(ticks []*Tick) error

	GetFeeRateByUserLevel(userLevel string) (*FeeRate, error)
}
