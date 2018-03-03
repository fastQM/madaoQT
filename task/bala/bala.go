package main

import (
	"sync"
	"time"

	"github.com/kataras/golog"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Task "madaoQT/task"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[TASK]")
}

type Config struct {
	IncreaseTrend bool
	Amount        float64
	Ratio         float64
	Step          float64
	LimitOpen     float64
}

type Position struct {
	Trade  Exchange.TradeType
	Open   float64
	Close  float64
	Amount float64
	Step   float64
}

type BalaTest struct {
	future        Exchange.IExchange
	status        Task.StatusType
	config        Config
	positions     map[uint]*Position
	positionIndex uint
}

func (p *BalaTest) Start(api string, secret string, configJSON string) error {
	Logger.Info("启动OKEx合约监视程序")

	futureExchange := new(Exchange.OKExAPI)
	futureExchange.SetConfigure(Exchange.Config{
		API:    api,
		Secret: secret,
		Custom: map[string]interface{}{
			"exchangeType": Exchange.ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	if err := futureExchange.Start(); err != nil {
		Logger.Errorf("Fail to start:%v", err)
		return err
	}

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					p.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventLostConnection {
					if p.status != Task.StatusNone && p.status != Task.StatusError {
						go Task.Reconnect(futureExchange)
					}
				}
			case <-time.After(5 * time.Second):
				if p.status == Task.StatusError || p.status == Task.StatusNone {
					Logger.Debug("状态异常或退出")
					return
				}

				if p.status == Task.StatusOrdering {
					Logger.Debug("交易中...")
					continue
				}

				p.Watch()
			}
		}
	}()

	return nil
}

var constContractRatio = map[string]float64{
	"btc": 100,
	"ltc": 10,
	"eth": 10,
}

func (p *BalaTest) Watch() {

	const pair = "eth/usdt"

	if p.positions == nil || len(p.positions) == 0 {

		var tradeType Exchange.TradeType
		var price float64

		err1, _, askFuturePlacePrice, _, bidFuturePlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.Amount)
		if err1 != nil {
			Logger.Debug("无效深度不操作")
			return
		}
		if p.config.IncreaseTrend {
			tradeType = Exchange.TradeTypeOpenLong
			price = askFuturePlacePrice
		} else {
			tradeType = Exchange.TradeTypeOpenShort
			price = bidFuturePlacePrice
		}

		futureConfig := Exchange.TradeConfig{
			Pair:   pair,
			Type:   tradeType,
			Price:  price,
			Amount: p.config.Amount / price,
			Limit:  p.config.LimitOpen,
		}

		channelFuture := Task.ProcessTradeRoutine(p.future, futureConfig, nil)
		var waitGroup sync.WaitGroup
		var futureResult Task.TradeResult
		waitGroup.Add(1)
		go func() {
			select {
			case futureResult = <-channelFuture:
				Logger.Debugf("合约交易结果:%v", futureResult)
				waitGroup.Done()
				if futureResult.Error != Task.TaskErrorSuccess {
					Logger.Errorf("建仓失败,请手工操作")
					p.status = Task.StatusError
					return
				} else {

					position := Position{
						Trade: tradeType,
						Open:  futureResult.AvgPrice,
						// Close:  futureResult.AvgPrice * (1 - p.config.Step*p.config.Ratio),
						Amount: futureResult.DealAmount,
						Step:   1,
					}

					p.positions[p.positionIndex] = &position
					p.positionIndex++
				}
			}
		}()

		waitGroup.Wait()

	} else {
		var placePrice float64
		for index, position := range p.positions {

			err1, _, askFuturePlacePrice, _, bidFuturePlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.Amount)
			if err1 != nil {
				Logger.Debug("无效深度不操作2")
				return
			}

			if position.Trade == Exchange.TradeTypeOpenLong {
				placePrice = bidFuturePlacePrice
			} else {
				placePrice = askFuturePlacePrice
			}

			//复利平仓再开仓
			if (position.Trade == Exchange.TradeTypeOpenLong && askFuturePlacePrice >= position.Open*(1+position.Step*p.config.Ratio)) ||
				(position.Trade == Exchange.TradeTypeOpenShort && bidFuturePlacePrice <= position.Open*(1-position.Step*p.config.Ratio)) {

				futureConfig := Exchange.TradeConfig{
					Pair:   pair,
					Type:   Exchange.RevertTradeType(position.Trade),
					Price:  placePrice,
					Amount: p.config.Amount / placePrice,
					Limit:  p.config.LimitOpen,
				}

				channelFuture := Task.ProcessTradeRoutine(p.future, futureConfig, nil)
				var waitGroup sync.WaitGroup
				var futureResult Task.TradeResult
				waitGroup.Add(1)
				go func() {
					select {
					case futureResult = <-channelFuture:
						Logger.Debugf("合约交易结果:%v", futureResult)
						waitGroup.Done()
					}
				}()

				waitGroup.Wait()

				if futureResult.Error != Task.TaskErrorSuccess {
					Logger.Errorf("平仓失败,请手工操作")
					p.status = Task.StatusError
					return
				}

				if position.Step == 3 {
					delete(p.positions, index)
					return
				} else {

					err1, _, askFuturePlacePrice, _, bidFuturePlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.Amount)
					if err1 != nil {
						Logger.Debug("无效深度不操作3")
						return
					}

					if position.Trade == Exchange.TradeTypeOpenLong {
						placePrice = askFuturePlacePrice
					} else {
						placePrice = bidFuturePlacePrice
					}

					p.positions[index].Step++
					futureConfig = Exchange.TradeConfig{
						Pair:   pair,
						Type:   position.Trade,
						Price:  placePrice,
						Amount: futureResult.AvgPrice * futureResult.DealAmount / placePrice,
						Limit:  p.config.LimitOpen,
					}

					channelFuture := Task.ProcessTradeRoutine(p.future, futureConfig, nil)
					var waitGroup sync.WaitGroup
					var futureResult Task.TradeResult
					waitGroup.Add(1)
					go func() {
						select {
						case futureResult = <-channelFuture:
							Logger.Debugf("合约交易结果:%v", futureResult)
							waitGroup.Done()
						}
					}()

					waitGroup.Wait()

					if futureResult.Error != Task.TaskErrorSuccess {
						Logger.Errorf("开仓失败,请手工操作")
						p.status = Task.StatusError
						return
					}
				}
				// 止损平仓
			} else if (position.Trade == Exchange.TradeTypeOpenLong && askFuturePlacePrice <= position.Open*(1-p.config.Step*p.config.Ratio)) ||
				(position.Trade == Exchange.TradeTypeOpenShort && bidFuturePlacePrice >= position.Open*(1+p.config.Step*p.config.Ratio)) {

				futureConfig := Exchange.TradeConfig{
					Pair:   pair,
					Type:   Exchange.RevertTradeType(position.Trade),
					Price:  placePrice,
					Amount: p.config.Amount / placePrice,
					Limit:  p.config.LimitOpen,
				}

				channelFuture := Task.ProcessTradeRoutine(p.future, futureConfig, nil)
				var waitGroup sync.WaitGroup
				var futureResult Task.TradeResult
				waitGroup.Add(1)
				go func() {
					select {
					case futureResult = <-channelFuture:
						Logger.Debugf("合约交易结果:%v", futureResult)
						waitGroup.Done()
					}
				}()

				waitGroup.Wait()

				if futureResult.Error != Task.TaskErrorSuccess {
					Logger.Errorf("平仓失败,请手工操作")
					p.status = Task.StatusError
					return
				} else {
					delete(p.positions, index)
				}
			}
		}
	}

}

func main(){
	
}