package exchange

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	Global "madaoQT/config"

	"github.com/kataras/golog"
)

// Logger: the handler of the module global logger
var logger *golog.Logger

func init() {
	_logger := golog.New()
	logger = _logger
	logger.SetLevel("debug")
	logger.SetTimeFormat(Global.TimeFormat)
	logger.SetPrefix("[EXCH]")
}

type ExchangeIndex int8

const (
	ExchangeOkex ExchangeIndex = iota
	ExchangeBinance
)

var ExchangeNameList = map[ExchangeIndex]string{
	ExchangeOkex:    NameOKEXSpot,
	ExchangeBinance: NameBinance,
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

	// EventLostConnection the event that the connection is in lost
	EventLostConnection EventType = iota

	// EventConnected the event that the exchange is connected
	EventConnected

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

func TradeTypeInt(tradeType string) TradeType {

	for i := 0; i < int(TradeTypeUnknown); i++ {
		if TradeTypeString[TradeType(i)] == tradeType {
			return TradeType(i)
		}
	}

	return -1
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
	OrderStatusRejected
	OrderStatusExpired
	OrderStatusOrdering
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
	// whether to use proxy, eg: "SOCKS5:127.0.0.1:1080"
	Proxy string
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

type KlineValue struct {
	Time      string
	OpenTime  float64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volumn    float64
	CloseTime float64
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
	Error     error
	ErrorCode int
	OrderID   string
	Info      *OrderInfo
}

// DepthTypeBids in which the prices should be from high to low
const DepthTypeBids = 0

// DepthTypeAsks in which the prices should be from low to high
const DepthTypeAsks = 1

/* 获取深度价格 */
type DepthPrice struct {
	Price    float64
	Quantity float64
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

	GetKline(pair string, period int, limit int) []KlineValue
}

// 以分钟为单位
const KlinePeriod5Min = 5
const KlinePeriod10Min = 10
const KlinePeriod15Min = 15
const KlinePeriod30Min = 30
const KlinePeriod1Hour = 60
const KlinePeriod2Hour = 120
const KlinePeriod4Hour = 240
const KlinePeriod6Hour = 6 * 60
const KlinePeriod12Hour = 12 * 60
const KlinePeriod1Day = 1 * 24 * 60
const KlinePeriodWeek = 0

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

func revertDepthArray(array []DepthPrice) []DepthPrice {
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

func GetArea(values []KlineValue) (float64, float64) {
	var high, low float64
	for _, kline := range values {
		if high == 0 {
			high = kline.High
		} else if kline.High > high {
			high = kline.High
		}

		if low == 0 {
			low = kline.Low
		} else if kline.Low < low {
			low = kline.Low
		}
	}

	return high, low
}

func GetAverage(period int, values []KlineValue) float64 {

	if values == nil || len(values) != period {
		log.Print("Error:Invalid values")
		return 0
	}

	var total float64
	for _, value := range values {
		total += value.Close
	}

	return total / float64(period)

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

// Example: ETH/USDT
func ParsePair(pair string) []string {
	return strings.Split(pair, "/")
}

const Path = "C:\\history\\"

func SaveHistory(code string, klines []KlineValue) {

	file, err := os.Create(Path + code + ".txt")
	if err != nil {
		log.Printf("Error1:%v", err)
		return
	}
	defer file.Close()

	for _, kline := range klines {
		data, err := json.Marshal(kline)
		if err != nil {
			log.Printf("Error2:%v", err)
			return
		}
		file.WriteString(string(data) + "\n")
	}
}

func LoadHistory(code string) []KlineValue {
	datas, err := ioutil.ReadFile(Path + code + ".txt")
	if err != nil {
		log.Printf("Error3:%v", err)
		return nil
	}

	var klines []KlineValue
	lines := strings.Split(string(datas), "\n")
	for _, line := range lines {
		if line != "" {
			var kline KlineValue
			// line = strings.Replace(line, "\n", "", 1)
			// log.Printf("line:%s", line)
			err := json.Unmarshal([]byte(line), &kline)
			if err != nil {
				log.Printf("Error4:%v", err)
				return nil
			}

			klines = append(klines, kline)
		}
	}

	return klines
}

func SaveHistoryWithSub(subdir string, code string, klines []KlineValue) {

	fullpath := Path
	if subdir != "" {
		fullpath = fullpath + subdir + "\\"
	}

	file, err := os.Create(fullpath + code + ".txt")
	if err != nil {
		log.Printf("Error1:%v", err)
		return
	}
	defer file.Close()

	for _, kline := range klines {
		data, err := json.Marshal(kline)
		if err != nil {
			log.Printf("Error2:%v", err)
			return
		}
		file.WriteString(string(data) + "\n")
	}
}

func LoadHistoryWithSub(subdir string, code string) []KlineValue {

	fullpath := Path
	if subdir != "" {
		fullpath = fullpath + subdir + "\\"
	}

	datas, err := ioutil.ReadFile(fullpath + code + ".txt")
	if err != nil {
		log.Printf("Error3:%v", err)
		return nil
	}

	var klines []KlineValue
	lines := strings.Split(string(datas), "\n")
	for _, line := range lines {
		if line != "" {
			var kline KlineValue
			// line = strings.Replace(line, "\n", "", 1)
			// log.Printf("line:%s", line)
			err := json.Unmarshal([]byte(line), &kline)
			if err != nil {
				log.Printf("Error4:%v", err)
				return nil
			}

			klines = append(klines, kline)
		}
	}

	return klines
}

func RevertArray(array []KlineValue) []KlineValue {
	var tmp KlineValue
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

func GetThreshHold(avg5first float64, avg5 float64, avg10first float64, avg10 float64) float64 {
	// 45x > 50*avg10 - 5 * avg10first - avg5 *50 + 10*avg5first
	return (50*avg10 - 5*avg10first - avg5*50 + 10*avg5first) / 5
}

func GetThreshHoldByAverage(avg1first float64, avg1 float64, interval1 float64, avg2first float64, avg2 float64, interval2 float64) float64 {
	// 45x > 50*avg10 - 5 * avg10first - avg5 *50 + 10*avg5first
	return (interval1*interval2*avg2 - interval1*avg2first - interval1*interval2*avg1 + interval2*avg1first) / (interval2 - interval1)
}

func CTPDailyKlinesToWeek(klines []KlineValue) []KlineValue {
	var KlinesByWeek []KlineValue

	var high, low, open, close, volumn float64
	var klineTime time.Time

	location, _ := time.LoadLocation("Asia/Shanghai")

	for _, kline := range klines {

		klineTime, _ = time.ParseInLocation("2006-01-02", kline.Time, location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close
		volumn += kline.Volumn

		if klineTime.Weekday() == time.Friday {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:   high,
					Low:    low,
					Open:   open,
					Close:  close,
					Time:   klineTime.Format("2006-01-02"),
					Volumn: volumn,
				}
				KlinesByWeek = append(KlinesByWeek, lastKline)
			}
			open = 0
			high = 0
			low = 0
			close = 0
			volumn = 0
		}

	}

	if high != 0 && low != 0 && close != 0 {
		lastKline := KlineValue{
			High:   high,
			Low:    low,
			Open:   open,
			Close:  close,
			Time:   klineTime.Format("2006-01-02"),
			Volumn: volumn,
		}
		KlinesByWeek = append(KlinesByWeek, lastKline)
	}

	return KlinesByWeek
}

func CTPDailyKlinesToMonth(klines []KlineValue) []KlineValue {
	var KlinesByMonth []KlineValue

	var high, low, open, close float64
	var klineTime time.Time
	var currentMonth time.Month

	location, _ := time.LoadLocation("Asia/Shanghai")
	first, _ := time.ParseInLocation("2006-01-02", klines[0].Time, location)
	currentMonth = first.Month()

	for i, kline := range klines {

		klineTime, _ = time.ParseInLocation("2006-01-02", kline.Time, location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close

		if i+1 < len(klines) {
			nextTime, _ := time.ParseInLocation("2006-01-02", klines[i+1].Time, location)
			if nextTime.Month() != currentMonth {
				if high != 0 && low != 0 && close != 0 {
					lastKline := KlineValue{
						High:  high,
						Low:   low,
						Open:  open,
						Close: close,
						Time:  klineTime.Format("2006-01-02"),
					}
					KlinesByMonth = append(KlinesByMonth, lastKline)
				}
				currentMonth = nextTime.Month()
				open = 0
				high = 0
				low = 0
				close = 0
			}
		} else {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:  high,
					Low:   low,
					Open:  open,
					Close: close,
					Time:  klineTime.Format("2006-01-02"),
				}
				KlinesByMonth = append(KlinesByMonth, lastKline)
			}
		}

	}

	return KlinesByMonth

}

func CTPDailyKlinesSplitToYears(klines []KlineValue) [][]KlineValue {
	var KlinesByYear [][]KlineValue
	var tmp []KlineValue
	var currentYear int

	location, _ := time.LoadLocation("Asia/Shanghai")
	first, _ := time.ParseInLocation("2006-01-02", klines[0].Time, location)
	currentYear = first.Year()

	for i, kline := range klines {
		tmp = append(tmp, kline)
		if i+1 < len(klines) {
			nextTime, _ := time.ParseInLocation("2006-01-02", klines[i+1].Time, location)
			if nextTime.Year() != currentYear {
				if tmp != nil && len(tmp) > 0 {
					KlinesByYear = append(KlinesByYear, tmp)
				}
				currentYear = nextTime.Year()
				tmp = nil
			}
		} else {
			if tmp != nil && len(tmp) > 0 {
				KlinesByYear = append(KlinesByYear, tmp)
			}

		}

	}

	return KlinesByYear

}
