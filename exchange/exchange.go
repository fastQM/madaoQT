package exchange

import (
	"strings"

	"github.com/kataras/golog"
)

// Logger: the handler of the module global logger
var logger *golog.Logger

func init() {
	_logger := golog.New()
	logger = _logger
	logger.SetLevel("debug")
	logger.SetTimeFormat("2006-01-02 06:04:05")
}

// ExchangeType the type of the exchange, spot or the future exchange
type ExchangeType int8

// EventType the type of the event, for example, notify the application that the connection with the exchange is lost
type EventType int8

// TradeType the type of the trading, for example: buy, sell, openlong, openshort, closelong, closeshort
type TradeType int8

// OrderStatusType the type of the order status, for example: open, partial-done, done
type OrderStatusType int8

const (
	// ExchangeTypeFuture the exchange for future-contract trading
	ExchangeTypeFuture ExchangeType = iota
	// ExchangeTypeSpot the exchange for spot trading
	ExchangeTypeSpot
)

const (
	// EventConnected the event that the exchange is connected
	EventConnected EventType = iota
	// EventLostConnection the event that the connection is in lost
	EventLostConnection
	// EventNum the common error
	EventNum
)

const (
	// TradeTypeOpenLong the OpenLong trade type of the future
	TradeTypeOpenLong TradeType = iota
	// TradeTypeOpenShort the OpenShort trade type of the future
	TradeTypeOpenShort
	// TradeTypeCloseLong the CloseLong trade type of the future
	TradeTypeCloseLong
	// TradeTypeCloseShort the CloseShort trade type of the future
	TradeTypeCloseShort
	// TradeTypeBuy the buy trade of the spot
	TradeTypeBuy
	// TradeTypeSell the sell trade of the spot
	TradeTypeSell
	// TradeTypeCancel the cancel trade of the future/spot
	TradeTypeCancel
	// TradeTypeUnknown the error type
	TradeTypeUnknown
)

// TradeTypeString the string description of the trade type
var TradeTypeString = map[TradeType]string{
	TradeTypeOpenLong:   "OpenLong",
	TradeTypeOpenShort:  "OpenShort",
	TradeTypeCloseLong:  "CloseLong",
	TradeTypeCloseShort: "CloseShort",
	TradeTypeBuy:        "Buy",
	TradeTypeSell:       "Sell",
	TradeTypeCancel:     "cancel",
	TradeTypeUnknown:    "Unknown_TradeType",
}

const (
	// OrderStatusOpen the open status of an order
	OrderStatusOpen OrderStatusType = iota
	// OrderStatusPartDone the part-done status of an order
	OrderStatusPartDone
	// OrderStatusDone the done status of an order
	OrderStatusDone
	// OrderStatusCanceling the canceling status of an order
	OrderStatusCanceling
	// OrderStatusCanceled the canceled status of an order
	OrderStatusCanceled
	// OrderStatusUnknown the error status
	OrderStatusUnknown
)

// OrderStatusString the string description of the status of the order
var OrderStatusString = map[OrderStatusType]string{
	OrderStatusOpen:      "Open",
	OrderStatusPartDone:  "PartDone",
	OrderStatusDone:      "Done",
	OrderStatusCanceling: "Canceling",
	OrderStatusCanceled:  "Canceled",
	OrderStatusUnknown:   "Unknown_OrderStatus",
}

// Config the configuration of the exchange
type Config struct {
	// API the api key of the exchange
	API string
	// Secret the secret key of the exchange
	Secret string
	// Ticker the applicatio should implement this interface to receive the information of the ticker of the exchange
	Ticker ITicker
	// custom configuration of the exchange
	Custom map[string]interface{}
}

type ITicker interface {
	Ticker(exchange string, pair string, value TickerValue)
}

type TickerListItem struct {
	// Pair used to get the ticker of the corresponding the pair/coin
	Pair string
	// Name the symbol in the exchange
	Symbol string
	Time   string
	Period string
	Value  interface{}
}

type TickerValue struct {
	High   float64
	Low    float64
	Volume float64
	Last   float64
	Time   string
	Period string // 合约周期
}

type TradeConfig struct {
	Batch string
	Pair  string

	/* buy or sell */
	Type   TradeType
	Price  float64
	Amount float64
	Limit  float64
}

type OrderInfo struct {
	Pair       string
	OrderID    string
	Price      float64
	Amount     float64
	DealAmount float64
	AvgPrice   float64
	Type       TradeType
	Status     OrderStatusType
}

type TradeResult struct {
	Error   error
	OrderID string
}

const DepthTypeBids = 0
const DepthTypeAsks = 1
/* 获取深度价格 */
type DepthPrice struct {
	Price float64
	Quantity   float64
}

// IExchange the interface of a exchange
type IExchange interface {
	// GetExchangeName() the function to get the name of the exchange
	GetExchangeName() string
	// SetConfigure()
	SetConfigure(config Config)
	// WatchEvent() return a channel which notified the application of the event triggered by exchange
	WatchEvent() chan EventType

	// Start() prepare the connection to the exchange
	Start() error
	// Close() close the connection to the exchange and other handles
	Close()

	// StartTicker() send message to the exchange to start the ticker of the given pairs
	StartTicker(pair string)
	// GetTicker(), better to use the ITicker to notify the ticker information
	GetTicker(pair string) *TickerValue

	// GetDepthValue() get the depth of the assigned price area and quantity
	// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
	GetDepthValue(pair string) [][]DepthPrice
	// GetBalance() get the balances of all the coins
	GetBalance() map[string]interface{}

	// Trade() trade as the configs
	Trade(configs TradeConfig) *TradeResult
	// CancelOrder() cancel the order as the order information
	CancelOrder(order OrderInfo) *TradeResult
	// GetOrderInfo() get the information with order filter
	GetOrderInfo(filter OrderInfo) []OrderInfo
}

// RevertTradeType the "close" operation of the original trading
func RevertTradeType(tradeType TradeType) TradeType {
	switch tradeType {
	case TradeTypeOpenLong:
		return TradeTypeCloseLong
	case TradeTypeOpenShort:
		return TradeTypeCloseShort
	case TradeTypeBuy:
		return TradeTypeSell
	case TradeTypeSell:
		return TradeTypeBuy
	}

	return TradeTypeUnknown
}


/*
	实际意义不大
*/
// func GetDepthAveragePrice(items []DepthPrice) (float64, float64) {

// 	if items == nil || len(items) == 0 {
// 		return -1, -1
// 	}

// 	var total float64
// 	var quantity float64

// 	for _, item := range items {
// 		total += item.price * item.qty
// 		quantity += item.qty
// 	}

// 	return total / quantity, quantity
// }

/*
	返回：（下单均价，下单价格）
*/
// func GetDepthPriceByOrder(items []DepthPrice, orderQty float64) (float64, float64) {
// 	if items == nil || len(items) == 0 {
// 		return -1, -1
// 	}

// 	// log.Printf("Depth:%v", items)
// 	var total float64
// 	var amount float64
// 	for _, item := range items {
// 		total += item.qty
// 		amount += (item.qty * item.price)
// 	}

// 	if orderQty > total {
// 		log.Printf("深度不够：%v", total)
// 		return amount / total, -2

// 	}

// 	var depth int
// 	balance := orderQty

// 	for i, item := range items {
// 		if balance-item.qty <= 0 {
// 			depth = i
// 			break
// 		} else {
// 			balance -= item.qty
// 		}
// 	}

// 	total = 0
// 	for i := 0; i < depth; i++ {
// 		total += items[i].price * items[i].qty
// 	}

// 	total += (items[depth].price * balance)

// 	return total / orderQty, items[depth].price
// }

func GetRatio(value1 float64, value2 float64) float64 {

	var big, small float64

	if value1 >= value2 {
		big = value1
		small = value2
	} else {
		big = value2
		small = value1
	}

	return (big - small) * 100 / small ///????
}

// Example: ETH/USDT
func ParsePair(pair string) []string {
	return strings.Split(pair, "/")
}
