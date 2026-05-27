package models

const (
	TopicOrder  = "g_order"   // Redis pub/sub topic name, not database table name   // Redis pub/sub topic name, not database table name
	TopicAccount = "g_account"
	TopicTrade   = "g_trade"
	TopicFill    = "g_fill"
	TopicBill    = "g_bill"
)

const (
	TableOrderSplitCount = 128
)
