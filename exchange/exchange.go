package exchange

import (
	"log"

	"github.com/kataras/golog"
)

/*
	初始化日志句柄
*/
var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
}

/* 交易类型：买，卖，开多，开空，平多，平空 */
type TradeType int8
type EventType int8
type OrderType int8
type OrderStatusType int8

const (
	TradeTypeFuture TradeType = iota
	TradeTypeSpot
)

const (
	EventConnected EventType = iota
	EventError
)

const (
	OrderTypeOpenLong OrderType = iota
	OrderTypeOpenShort
	OrderTypeCloseLong
	OrderTypeCloseShort
	OrderTypeBuy
	OrderTypeSell
	OrderTypeUnknown
)

const (
	OrderStatusOpen OrderStatusType = iota
	OrderStatusPartDone
	OrderStatusDone
	OrderStatusCanceling
	OrderStatusCanceled
	OrderStatusUnknown
)

type InitConfig struct {
	Api    string
	Secret string
	Custom map[string]interface{}
}

type TickerListItem struct {
	Tag    string // 用户调用者匹配
	Name   string // 用户交易所匹配
	Time   string
	Type   TradeType // 合约还是现货
	Period string    // 合约周期
	Value  interface{}

	ticket    int64
	oldticket int64
}

type DepthListItem struct {
	Coin       string
	Name       string
	Time       string
	Depth      string
	AskAverage float64
	AskQty     float64
	BidAverage float64
	BidQty     float64
	AskByOrder float64
	BidByOrder float64
}

type DepthValue struct {
	Time       string
	AskAverage float64
	AskQty     float64
	BidAverage float64
	BidQty     float64
	AskByOrder float64 // 下单深度均价
	AskPrice   float64 // 下单价格
	BidByOrder float64
	BidPrice   float64
}

type TickerValue struct {
	Last   float64
	Time   string
	Type   TradeType
	Period string // 合约周期
}

type TradeConfig struct {
	Coin string
	/* buy or sell */
	Type   OrderType
	Price  float64
	Amount float64
}

type OrderInfo struct {
	Coin    string
	OrderID string
	Price   float64
	Amount  float64
	Type    OrderType
	Status  OrderStatusType
}

type TradeResult struct {
	Error   error
	OrderID string
}

/* 获取深度价格 */
type DepthPrice struct {
	price float64
	qty   float64
}

type IExchange interface {
	GetExchangeName() string
	Init(config InitConfig)
	Start()
	// AddTicker(coinA string, coinB string, config interface{}, tag string)
	GetTickerValue(tag string) *TickerValue
	WatchEvent() chan EventType
	GetDepthValue(coinA string, coinB string, orderQuantity float64) *DepthValue
	GetBalance(coin string) float64

	Trade(configs TradeConfig) *TradeResult

	CancelOrder(order OrderInfo) map[string]interface{}
	GetOrderInfo(filter map[string]interface{}) []OrderInfo
}

const DepthTypeAsks = 0
const DepthTypeBids = 1

func RevertDepthArray(array []DepthPrice) []DepthPrice {
	var tmp DepthPrice
	var length int

	if len(array)%2 != 0 {
		length = len(array) / 2
	} else {
		length = len(array)/2 - 1
	}
	for i := 0; i <= length; i++ {
		tmp = array[i]
		array[i] = array[len(array)-1-i]
		array[len(array)-1-i] = tmp

	}
	return array
}

func GetDepthAveragePrice(items []DepthPrice) (float64, float64) {

	if items == nil || len(items) == 0 {
		return -1, -1
	}

	var total float64
	var quantity float64

	for _, item := range items {
		total += item.price * item.qty
		quantity += item.qty
	}

	return total / quantity, quantity
}

func GetDepthPriceByOrder(depthType int, items []DepthPrice, orderQty float64) (float64, float64) {
	if items == nil || len(items) == 0 {
		return -1, -1
	}

	if depthType == DepthTypeAsks {
		// 倒序
		items = RevertDepthArray(items)
	}
	// log.Printf("Depth:%v", items)
	var total float64
	for _, item := range items {
		total += item.qty
	}

	if orderQty > total {
		log.Printf("深度不够：%v", total)
		return -2, -2
	}

	var depth int
	balance := orderQty

	for i, item := range items {
		if balance-item.qty <= 0 {
			depth = i
			break
		} else {
			balance -= item.qty
		}
	}

	total = 0
	for i := 0; i < depth; i++ {
		total += items[i].price * items[i].qty
	}

	total += (items[depth].price * balance)

	return total / orderQty, items[depth].price
}

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

type Exchanges struct {
	exchanges map[string]IExchange
}

// func (e *Exchanges) Init() {
// 	/* add exchange list */
// 	okexfuture := new(OKExAPI)
// 	okexfuture.Init(InitConfig{
// 		Api:    constOKEXApiKey,
// 		Secret: constOEXSecretKey,
// 		Custom: map[string]interface{}{"tradeType": TradeTypeFuture},
// 	})

// 	e.exchanges[okexfuture.GetExchangeName()] = okexfuture

// 	okexspot := new(OKExAPI)
// 	okexspot.Init(InitConfig{
// 		Api:    constOKEXApiKey,
// 		Secret: constOEXSecretKey,
// 		Custom: map[string]interface{}{"tradeType": TradeTypeSpot},
// 	})

// 	e.exchanges[okexfuture.GetExchangeName()] = okexspot
// }

// func (e *Exchanges) Start() {

// }
