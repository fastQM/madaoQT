package task

import (
	"errors"
	"math"
	"time"

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
}

func ProcessTradeRoutine(exchange Exchange.IExchange,
	tradeConfig Exchange.TradeConfig,
	dbTrades *Mongo.Trades) chan TradeResult {

	// var balance interface{} // 实际余额后台返回为准
	// coin := Exchange.ParsePair(tradeConfig.Coin)[0]
	channel := make(chan TradeResult)
	stopTime := time.Now().Add(10 * time.Second)

	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)

		// var dealAmount, totalCost, avePrice float64
		var dealAmount, totalCost, avePrice float64
		var trade *Exchange.TradeResult
		// var depthInvalidCount int
		var errorCode TaskErrorType
		// var depth *Exchange.DepthValue
		// var depth [][]Exchange.DepthPrice
		var tradePrice, tradeAmount float64
		// var err error

		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("Timeout when trading")
				errorCode = TaskErrorTimeout
				goto __ERROR
			}

			// 1. 根据深度情况计算价格下单
			// depth = exchange.GetDepthValue(tradeConfig.Pair)

			// if depth == nil {
			// 	Logger.Debugf("无操作价格")
			// 	depthInvalidCount++
			// 	/*
			// 		连续十次无法达到操作价格，则退出
			// 	*/
			// 	if depthInvalidCount > 10 {
			// 		errorCode = TaskInvalidDepth
			// 		goto __ERROR
			// 	}
			// 	goto _NEXTLOOP
			// } else {
			// 	Logger.Debugf("深度:%v", depth)
			// 	err, tradePrice, tradeAmount = getPlacedPrice(tradeConfig.Type,
			// 		depth,
			// 		tradeConfig.Price,
			// 		tradeConfig.Limit,
			// 		tradeConfig.Amount-dealAmount)

			// 	if err != nil {
			// 		Logger.Errorf("Trade Error:%v", err)
			// 		Utils.SleepAsyncBySecond(3)
			// 		goto _NEXTLOOP
			// 	}

			// 	Logger.Debugf("交易价格：%v 交易数量:%v", tradePrice, tradeAmount)
			// }

			// depthInvalidCount = 0

			tradePrice = getPlacedPrice(tradeConfig.Type, tradeConfig.Price, tradeConfig.Limit)

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
					Logger.Errorf("Fail to save trade record:%v", err)
				}
			}

			if trade != nil && trade.Error == nil {
				// 300 seconds = 5 minutes
				loop := 20

				for {
					Utils.SleepAsyncByMillisecond(500)

					info := exchange.GetOrderInfo(Exchange.OrderInfo{
						OrderID: trade.OrderID,
						Pair:    tradeConfig.Pair,
					})

					if info == nil || len(info) == 0 {
						Logger.Error("Fail to get the order info")
						goto __ERROR
					}

					// dbOrders.Insert(&Mongo.OrderInfo{
					// 	Batch:    tradeConfig.Batch,
					// 	Exchange: exchange.GetExchangeName(),
					// 	Coin:     tradeConfig.Coin,
					// 	OrderID:  trade.OrderID,
					// 	Status:   Exchange.OrderStatusString[info[0].Status],
					// })

					if info[0].Status == Exchange.OrderStatusDone {
						dealAmount += info[0].DealAmount
						totalCost += (info[0].AvgPrice * info[0].DealAmount) //手续费如何？
						if dbTrades != nil {
							dbTrades.SetDone(trade.OrderID)
						}
						goto __CheckDealAmount
					}

					loop--
					Logger.Debugf("Waiting for the trading result...")

					if loop == 0 {
						Logger.Debugf("Timeout，cancel the order...")
						// cancle the order, if it is traded when we cancle?
						trade := exchange.CancelOrder(Exchange.OrderInfo{
							Pair:    tradeConfig.Pair,
							OrderID: info[0].OrderID,
						})

						// if err := dbTrades.Insert(&Mongo.TradesRecord{
						// 	Batch:   tradeConfig.Batch,
						// 	Oper:    Exchange.TradeTypeString[Exchange.TradeTypeCancel],
						// 	OrderID: trade.OrderID,
						// 	// Details: fmt.Sprintf("%v", trade),
						// }); err != nil {
						// 	Logger.Errorf("保存交易操作失败:%v", err)
						// }

						if trade != nil && trade.Error == nil {

							info := exchange.GetOrderInfo(Exchange.OrderInfo{
								OrderID: trade.OrderID,
								Pair:    tradeConfig.Pair,
							})

							if info == nil || len(info) == 0 {
								Logger.Error("Fail to get the order info")
								goto __ERROR
							}

							if dbTrades != nil {
								dbTrades.SetCanceled(trade.OrderID)
							}

							// dbOrders.Insert(&Mongo.OrderInfo{
							// 	Batch:    tradeConfig.Batch,
							// 	Exchange: exchange.GetExchangeName(),
							// 	Coin:     tradeConfig.Coin,
							// 	OrderID:  trade.OrderID,
							// 	Status:   Exchange.OrderStatusString[info[0].Status],
							// 	// Details:  fmt.Sprintf("%v", info[0]),
							// })

							dealAmount += info[0].DealAmount
							totalCost += (info[0].AvgPrice * info[0].DealAmount)
							Logger.Debugf("Succeed to get the order info:%v, deal amout:%v", info[0].OrderID, dealAmount)
							goto __CheckDealAmount

						} else {
							Logger.Errorf("Fail to cancel the order:%v", info[0].OrderID)
							errorCode = TaskUnableCancelOrder
							goto __ERROR
						}
					}
				}
			} else {
				errorCode = TaskUnableTrade
				goto __ERROR
			}

		__ERROR:
			if dealAmount != 0 {
				avePrice = totalCost / dealAmount
			}

			channel <- TradeResult{
				Error:      errorCode,
				DealAmount: dealAmount,
				AvgPrice:   avePrice,
			}

			return
		__CheckBalance:
			if dealAmount != 0 {
				avePrice = totalCost / dealAmount
			}

			channel <- TradeResult{
				Error:      TaskErrorSuccess,
				AvgPrice:   avePrice,
				DealAmount: dealAmount,
			}
			return

		__CheckDealAmount:
			Logger.Debugf("Deal:%v Total:%v", dealAmount, tradeConfig.Amount)
			if tradeConfig.Amount-dealAmount >= 0.01 {
				goto _NEXTLOOP
			}
			// else
			goto __CheckBalance

		_NEXTLOOP:
			// 	延时
			Utils.SleepAsyncBySecond(1)
			continue
		}
	}()

	return channel

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

			tradePrice = getPlacedPrice(tradeConfig.Type, tradeConfig.Price, tradeConfig.Limit)

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

func getPlacedPrice(tradeType Exchange.TradeType, price float64, limit float64) float64 {

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
		return errors.New("交易所接口无效"), 0, 0, 0, 0
	}
	depths := exchange.GetDepthValue(pair)
	// Logger.Debugf("Future:%v 深度:%v", isFuture, depths)
	if depths != nil {
		asks = depths[Exchange.DepthTypeAsks]
		bids = depths[Exchange.DepthTypeBids]
	} else {
		return errors.New("未获取深度信息"), 0, 0, 0, 0
	}

	// amount *= 2

	if isFuture {
		quantity = amount / ratios[Exchange.ParsePair(pair)[0]]
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
