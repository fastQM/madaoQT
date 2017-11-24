package exchange

type TradeType int8
type EventType int8

const (
	TradeTypeContract TradeType = iota
	TradeTypeCurrent
)

const (
	EventConnected EventType = iota
	EventError
)

type TickerListItem struct{
	Tag string	// 用户调用者匹配
	Name string	// 用户交易所匹配
	Time string
	Type TradeType	// 合约还是现货
	Period string // 合约周期
	Value interface{}
}

type DepthItemValue struct {
	Price float64
	CoinQuantity float64
	AllQuantity float64
}

type DepthListItem struct {
	Tag string
	Name string
	Time string
	Asks []interface{}
	Bids []interface{}
}

type TickerValue struct {
	Last float64
	Time string
	Type TradeType
	Period string // 合约周期
}

type IExchange interface{
	GetExchangeName() string
	// Init(config interface{}) error
	// AddTicker(coinA string, coinB string, config interface{}, tag string)
	GetTickerValue(tag string) *TickerValue
	WatchEvent() chan EventType 	 
}

