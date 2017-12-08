package task

import (
	"errors"
	Exchange "madaoQT/exchange"
	"madaoQT/utils"
	"time"

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
	Logger.Info("Rules package init() finished")
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
	defer close(channel)

	stopTime := time.Now().Add(1 * time.Minute)

	go func() {
		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("超出操作时间")
				channel <- TradeResult{
					Error:      errors.New("timeout"),
					AmountLeft: tradeConfig.Amount,
				}
				return
			}

			var tradeAmount float64

			depth := exchange.GetDepthValue(tradeConfig.Coin, "", tradeConfig.Price, tradeConfig.Limit, tradeConfig.Amount, tradeConfig.Type)

			result := exchange.Trade(Exchange.TradeConfig{
				Coin:   tradeConfig.Coin,
				Type:   tradeConfig.Type,
				Amount: depth.LimitTradeAmount,
				Price:  depth.LimitTradePrice,
			})

			if result != nil && result.Error != nil {
				loop := 3
				for {
					utils.SleepAsyncBySecond(3)
					info := exchange.GetOrderInfo(map[string]interface{}{})
					if info[0].Status == Exchange.OrderStatusDone {
						tradeConfig.Amount -= tradeAmount
						if tradeConfig.Amount > 0.01 {
							goto _NEXTLOOP
						}
						// else
						channel <- TradeResult{
							Error: nil,
						}
						return
					}
					loop--
					if loop == 0 {
						// cancle the order, if it is traded when we cancle?
						result := exchange.CancelOrder(Exchange.OrderInfo{
							Coin:    tradeConfig.Coin,
							OrderID: info[0].OrderID,
						})

						Logger.Infof("Cancel order result:%v", result)

						goto _NEXTLOOP
					}
				}
			}

		_NEXTLOOP:
		}
	}()

	return channel

}
