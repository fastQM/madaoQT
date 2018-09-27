package task

import (
	"errors"
	"math"

	"github.com/kataras/golog"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	Utils "madaoQT/utils"
)

/*
	初始化日志句柄
*/
var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[TASK]")
}

type EventType int8
type TaskErrorType int

const (
	EventTypeError EventType = iota
	EventTypeTrigger
)

const (
	TaskErrorSuccess TaskErrorType = iota
	TaskErrorTimeout
	TaskInvalidDepth
	TaskUnableTrade
	TaskUnableGetOrderInfo
	TaskUnableCancelOrder
	TaskInvalidConfig
	TaskErrorStatus
	TaskLostMongodb
	TaskInvalidInput
	TaskAPINotFound
	TaskIOCReturn
)

var TaskErrorMsg = map[TaskErrorType]string{
	TaskErrorSuccess:      "success",
	TaskErrorTimeout:      "timeout",
	TaskInvalidDepth:      "Invalid Depth",
	TaskUnableTrade:       "Unable to trade",
	TaskUnableCancelOrder: "Unable to cancel order",
	TaskInvalidConfig:     "Invalid configure",
	TaskErrorStatus:       "Error status",
	TaskLostMongodb:       "Lost the connection of Mongodb",
	TaskInvalidInput:      "Invalid Input",
	TaskAPINotFound:       "API or Key not found",
}

type TradeResult struct {
	Error      TaskErrorType
	DealAmount float64 // 已成交金额，如果部分成交，需要将该部分平仓
	AvgPrice   float64
	OrderID    string
}

func ProcessTradeRoutineIOC(exchange Exchange.IExchange,
	tradeConfig Exchange.TradeConfig,
	dbTrades *Mongo.Trades) chan TradeResult {

	channel := make(chan TradeResult)
	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)

		var dealAmount, avgPrice float64
		var trade *Exchange.TradeResult
		var errorCode TaskErrorType

		var tradePrice, tradeAmount float64

		for {

			tradePrice = GetPlacedPrice(tradeConfig.Type, tradeConfig.Price, tradeConfig.Limit)

			trade = exchange.Trade(Exchange.TradeConfig{
				Pair:   tradeConfig.Pair,
				Type:   tradeConfig.Type,
				Amount: tradeConfig.Amount - dealAmount,
				Price:  tradePrice,
			})

			if dbTrades != nil {
				if err := dbTrades.Insert(&Mongo.TradesRecord{
					Batch:    tradeConfig.Batch,
					Oper:     Exchange.TradeTypeString[tradeConfig.Type],
					Exchange: exchange.GetExchangeName(),
					Pair:     tradeConfig.Pair,
					Quantity: tradeAmount,
					Price:    tradePrice,
					OrderID:  trade.OrderID,
				}); err != nil {
					Logger.Errorf("保存交易操作失败:%v", err)
				}
			}

			if trade != nil && trade.Error == nil {

				if trade.Info == nil {
					channel <- TradeResult{
						Error: TaskIOCReturn,
					}
					return
				} else {
					dealAmount = trade.Info.DealAmount
					avgPrice = trade.Info.AvgPrice

					if dbTrades != nil {
						dbTrades.SetDone(trade.OrderID)
					}
					channel <- TradeResult{
						Error:      TaskErrorSuccess,
						DealAmount: dealAmount,
						AvgPrice:   avgPrice,
					}
					return
				}

			} else {
				Logger.Errorf("交易失败：%v", trade.Error)
				errorCode = TaskUnableTrade
				channel <- TradeResult{
					Error: errorCode,
				}
				return
			}
		}
	}()

	return channel

}

func OutFuturePriceArea(futureConfig Exchange.TradeConfig, askPrice float64, bidPrice float64, area float64) bool {

	if area <= 0 {
		Logger.Errorf("Invalid Area:%v", area)
		return false
	}

	if futureConfig.Type == Exchange.TradeTypeCloseLong {
		if askPrice > futureConfig.Price*(1+area) {
			return true
		}
	} else if futureConfig.Type == Exchange.TradeTypeCloseShort {
		if bidPrice < futureConfig.Price*(1-area) {
			return true
		}
	}

	return false
}

func CheckPriceDiff(spotConfig Exchange.TradeConfig, futureConfig Exchange.TradeConfig,
	askFuturePrice float64, bidFuturePrice float64, askSpotPrice float64, bidSpotPrice float64, close float64) bool {

	if spotConfig.Type == Exchange.TradeTypeBuy && futureConfig.Type == Exchange.TradeTypeCloseLong {
		if math.Abs(askSpotPrice-bidFuturePrice)*100/bidFuturePrice < close {
			return true
		}
	} else if spotConfig.Type == Exchange.TradeTypeSell && futureConfig.Type == Exchange.TradeTypeCloseShort {
		if math.Abs(askFuturePrice-bidSpotPrice)*100/bidSpotPrice < close {
			return true
		}

	} else {
		Logger.Error("无效的交易配置")
		return false
	}

	return false
}

func GetPlacedPrice(tradeType Exchange.TradeType, price float64, limit float64) float64 {

	if tradeType == Exchange.TradeTypeOpenLong || tradeType == Exchange.TradeTypeCloseShort || tradeType == Exchange.TradeTypeBuy {
		limitPriceHigh := price * (1 + limit)
		Logger.Debugf("Buy Price:%v", limitPriceHigh)
		return limitPriceHigh
	} else {
		limitPriceLow := price * (1 - limit)
		Logger.Debugf("Sell Price:%v", limitPriceLow)
		return limitPriceLow

	}
}

func TranslateToContractNumber(price float64, coinQuantity float64) int {
	return int(coinQuantity * price / 10)
}

func CalcDepthPrice(isFuture bool, ratios map[string]float64, exchange Exchange.IExchange, pair string, amount float64) (err error, askPrice float64, askPlacePrice float64, bidPrice float64, bidPlacePrice float64) {

	var asks, bids []Exchange.DepthPrice
	var quantity float64
	var askFlag, bidFlag bool
	if exchange == nil {
		return errors.New("Invalid exchange"), 0, 0, 0, 0
	}
	depths := exchange.GetDepthValue(pair)
	// Logger.Debugf("Future:%v Depth:%v", isFuture, depths)
	if depths != nil {
		asks = depths[Exchange.DepthTypeAsks]
		bids = depths[Exchange.DepthTypeBids]
	} else {
		return errors.New("Fail to get depth info"), 0, 0, 0, 0
	}

	// amount *= 2

	if isFuture {
		quantity = amount / ratios[Exchange.ParsePair(pair)[0]]
		Logger.Debugf("Quantity:%f", quantity)
	}

	// Logger.Debugf("Asks:%v", asks)
	// Logger.Debugf("Bids:%v", bids)

	if asks != nil && len(asks) != 0 {
		var totalQuantity, totalAmount float64
		for _, depth := range asks {

			if isFuture {
				if (totalQuantity + depth.Quantity) >= quantity {

					totalAmount += depth.Price * (quantity - totalQuantity)
					totalQuantity = quantity
					askFlag = true
					askPrice = totalAmount / totalQuantity
					askPlacePrice = depth.Price
					break
				} else {
					totalQuantity += depth.Quantity
					totalAmount += depth.Quantity * depth.Price
				}
			} else {
				if (totalAmount + depth.Price*depth.Quantity) >= amount {
					totalQuantity += (amount - totalAmount) / depth.Price
					totalAmount = amount
					askFlag = true
					askPrice = totalAmount / totalQuantity
					askPlacePrice = depth.Price
					break
				} else {
					totalQuantity += depth.Quantity
					totalAmount += depth.Quantity * depth.Price
				}
			}

		}
	}

	if bids != nil && len(bids) != 0 {
		var totalQuantity, totalAmount float64
		for _, depth := range bids {
			if isFuture {
				if (totalQuantity + depth.Quantity) >= quantity {

					totalAmount += depth.Price * (quantity - totalQuantity)
					totalQuantity = quantity
					bidFlag = true
					bidPrice = totalAmount / totalQuantity
					bidPlacePrice = depth.Price
					break
				} else {
					totalQuantity += depth.Quantity
					totalAmount += depth.Quantity * depth.Price
				}
			} else {
				if (totalAmount + depth.Price*depth.Quantity) >= amount {
					totalQuantity += (amount - totalAmount) / depth.Price
					totalAmount = amount
					bidFlag = true
					bidPrice = totalAmount / totalQuantity
					bidPlacePrice = depth.Price
					break
				} else {
					totalQuantity += depth.Quantity
					totalAmount += depth.Quantity * depth.Price
				}
			}
		}
	}

	if askFlag && bidFlag {
		return nil, askPrice, askPlacePrice, bidPrice, bidPlacePrice
	}

	return errors.New("Invalid depth"), 0, 0, 0, 0
}

func Reconnect(exchange Exchange.IExchange) {
	Logger.Debug("Reconnecting......")
	Utils.SleepAsyncBySecond(10)
	if err := exchange.Start(); err != nil {
		Logger.Errorf("Fail to start exchange %v with error:%v", exchange.GetExchangeName(), err)
		return
	}
}
