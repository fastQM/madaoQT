package task

import Mongo "madaoQT/mongo"

/*
	静态加载的任务
*/

type StatusType int

const (
	StatusNone StatusType = iota
	StatusInit
	StatusProcessing
	StatusError
)

type Description struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

type Balance struct {
	Coin    string  `json:"coin"`
	Type    string  `json:"type"`
	Balance float64 `json:"float64"`
}

const BalanceTypeSpot = "spot"
const BalanceTypeFuture = "future"
const BalanceTypeBond = "bond"

type TradeRecord struct {
	Time     string  `json:"time"`
	Oper     string  `json:"oper"`
	Exchange string  `json:"exchange"`
	Pair     string  `json:"pair"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	OrderID  string  `json:"orderid"`
	Status   string  `json:"status"`
}

type ITask interface {

	// Task Management

	// GetDefaultConfig() get the default configuration of the task
	GetDefaultConfig() interface{}
	GetDescription() Description
	Start(configJSON string) error
	Close()
	GetStatus() StatusType

	// Fundation Management

	// GetBalances() []Balance
	GetBalances() map[string]interface{}
	GetTrades() []Mongo.TradesRecord
	// GetTrades() []TradeRecord
	GetPositions() []map[string]interface{}
	GetFailedPositions() []map[string]interface{}
	FixFailedPosition(updateJSON string) error
}
