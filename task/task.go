package task

import (
	"errors"
	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	"madaoQT/utils"
	"time"

	"github.com/kataras/golog"
)

var constOKEXApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
var constOEXSecretKey = "71430C7FA63A067724FB622FB3031970"

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

type EventType int8

const (
	EventTypeError EventType = iota
	EventTypeTrigger
)

type RulesEvent struct {
	EventType EventType
	Msg       interface{}
}

type TradeResult struct {
	Error    error
	Balance  float64 // 成交后余额
	AvgPrice float64 // 成交均价
}

func ProcessTradeRoutine(exchange Exchange.IExchange, tradeConfig Exchange.TradeConfig, db *Mongo.Trades) chan TradeResult {

	var balance float64 // 实际余额后台返回为准
	coin := Exchange.ParseCoins(tradeConfig.Coin)[0]
	channel := make(chan TradeResult)
	stopTime := time.Now().Add(1 * time.Minute)

	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)
		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("超出操作时间")
				channel <- TradeResult{
					Error:   errors.New("timeout"),
					Balance: tradeConfig.Amount,
				}
				return
			}

			var dealAmount, totalCost, avePrice float64
			var trade *Exchange.TradeResult

			// 1. 根据深度情况计算价格下单
			depth := exchange.GetDepthValue(tradeConfig.Coin, tradeConfig.Price, tradeConfig.Limit, tradeConfig.Amount, tradeConfig.Type)

			Logger.Debugf("深度信息:%v 下单信息：%v", depth, tradeConfig)

			if depth == nil || depth.LimitTradeAmount == 0 || depth.LimitTradePrice == 0 {
				Logger.Debugf("无操作价格:%v", depth)
				goto _NEXTLOOP
			}

			trade = exchange.Trade(Exchange.TradeConfig{
				Coin:   tradeConfig.Coin,
				Type:   tradeConfig.Type,
				Amount: depth.LimitTradeAmount,
				Price:  depth.LimitTradePrice,
			})

			if trade != nil && trade.Error == nil {

				if err := db.Insert(&Mongo.TradesRecord{
					Time:     time.Now(),
					Oper:     Exchange.GetTradeTypeString(tradeConfig.Type),
					Exchange: exchange.GetExchangeName(),
					Coin:     tradeConfig.Coin,
					Quantity: depth.LimitTradeAmount,
					Price:    depth.LimitTradePrice,
					OrderID:  trade.OrderID,
				}); err != nil {
					Logger.Errorf("保存交易操作失败:%v", err)
				}

				loop := 10

				for {
					utils.SleepAsyncBySecond(1)
					info := exchange.GetOrderInfo(Exchange.OrderInfo{
						OrderID: trade.OrderID,
						Coin:    tradeConfig.Coin,
					})
					if info[0].Status == Exchange.OrderStatusDone {
						dealAmount += info[0].DealAmount
						totalCost += (info[0].AvgPrice * info[0].DealAmount) //手续费如何？
						goto __CheckDealAmount
					}

					loop--
					Logger.Debugf("等待成交...")
					if loop == 0 {
						Logger.Debugf("超时，取消订单...")
						// cancle the order, if it is traded when we cancle?
						trade := exchange.CancelOrder(Exchange.OrderInfo{
							Coin:    tradeConfig.Coin,
							OrderID: info[0].OrderID,
						})

						if trade != nil && trade.Error == nil {

							info := exchange.GetOrderInfo(Exchange.OrderInfo{
								OrderID: trade.OrderID,
								Coin:    tradeConfig.Coin,
							})

							dealAmount += info[0].DealAmount
							totalCost += (info[0].AvgPrice * info[0].DealAmount)
							Logger.Debugf("成功取消订单：%v, 已成交金额:%v", info[0].OrderID, dealAmount)

							goto __CheckDealAmount

						} else {
							Logger.Errorf("取消订单：%v失败，请手动操作", info[0].OrderID)

							channel <- TradeResult{
								Error: errors.New("取消订单失败，操作异常"),
							}
							return
						}
					}
				}
			} else {
				Logger.Errorf("交易失败：%v", trade.Error)
				channel <- TradeResult{
					Error: errors.New("交易失败"),
				}
				return
			}

		__CheckBalance:
			balance = exchange.GetBalance(coin)
			avePrice = totalCost / dealAmount
			Logger.Debugf("交易完成，余额：%v 成交均价：%v", balance, avePrice)
			channel <- TradeResult{
				Error:    nil,
				Balance:  balance,
				AvgPrice: avePrice,
			}
			return

		__CheckDealAmount:
			Logger.Debugf("已成交:%v 总量:%v", dealAmount, tradeConfig.Amount)
			if tradeConfig.Amount-dealAmount > 0 {
				goto _NEXTLOOP
			}
			// else
			goto __CheckBalance

		_NEXTLOOP:
			// 	延时
			utils.SleepAsyncBySecond(1)
			continue
		}
	}()

	return channel

}

func InPriceArea(price float64, baseprice float64, area float64) bool {

	if area <= 0 {
		Logger.Errorf("Invalid Area:%v", area)
		return false
	}

	high := baseprice * (1 + area)
	low := baseprice * (1 - area)

	if price <= high && price >= low {
		return true
	}

	return false
}

func TranslateToContractNumber(price float64, coinQuantity float64) int {
	return int(coinQuantity * price / 10)
}
