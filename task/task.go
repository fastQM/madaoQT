package task

import (
	"errors"
	Exchange "madaoQT/exchange"
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
	Error      error
	AmountLeft float64
}

func ProcessTradeRoutine(exchange Exchange.IExchange, tradeConfig Exchange.TradeConfig) chan TradeResult {

	channel := make(chan TradeResult)
	stopTime := time.Now().Add(1 * time.Minute)

	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)
		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("超出操作时间")
				channel <- TradeResult{
					Error:      errors.New("timeout"),
					AmountLeft: tradeConfig.Amount,
				}
				return
			}

			var result *Exchange.TradeResult
			depth := exchange.GetDepthValue(tradeConfig.Coin, tradeConfig.Price, tradeConfig.Limit, tradeConfig.Amount, tradeConfig.Type)

			Logger.Debugf("Depth:%v", depth)

			if depth == nil || depth.LimitTradeAmount == 0 || depth.LimitTradePrice == 0 {
				Logger.Debugf("无操作价格:%v", depth)
				goto _NEXTLOOP
			}

			result = exchange.Trade(Exchange.TradeConfig{
				Coin:   tradeConfig.Coin,
				Type:   tradeConfig.Type,
				Amount: depth.LimitTradeAmount,
				Price:  depth.LimitTradePrice,
			})

			if result != nil && result.Error == nil {
				loop := 10
				for {
					utils.SleepAsyncBySecond(1)
					info := exchange.GetOrderInfo(map[string]interface{}{
						"order_id": result.OrderID,
						"symbol":   tradeConfig.Coin,
					})
					if info[0].Status == Exchange.OrderStatusDone {
						tradeConfig.Amount -= depth.LimitTradeAmount
						Logger.Debugf("成交:%v 未成交:%v", depth.LimitTradeAmount, tradeConfig.Amount)
						if tradeConfig.Amount > 0 {
							goto _NEXTLOOP
						}
						// else
						Logger.Debug("交易完成")
						channel <- TradeResult{
							Error: nil,
						}
						return
					}

					loop--
					Logger.Debugf("等待成交...")
					if loop == 0 {
						Logger.Debugf("超时，取消订单...")
						// cancle the order, if it is traded when we cancle?
						result := exchange.CancelOrder(Exchange.OrderInfo{
							Coin:    tradeConfig.Coin,
							OrderID: info[0].OrderID,
						})

						if result != nil && result.Error == nil {
							Logger.Debugf("成功取消订单：%v", info[0].OrderID)
						} else {
							Logger.Errorf("取消订单：%v失败，请手动操作", info[0].OrderID)
							channel <- TradeResult{
								Error: errors.New("取消订单失败，操作异常"),
							}
							return
						}

						goto _NEXTLOOP
					}
				}
			} else {
				Logger.Errorf("交易失败：%v", result)
				channel <- TradeResult{
					Error: errors.New("交易失败"),
				}
				return
			}

		_NEXTLOOP:
			// 	延时
			utils.SleepAsyncBySecond(1)
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
