package task

import (
	"errors"
	"math"
	"os"
	"os/exec"
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
}

type TradeResult struct {
	Error      TaskErrorType
	DealAmount float64 // 已成交金额，如果部分成交，需要将该部分平仓
	AvgPrice   float64
}

type TaskExplanation struct {
	Name        string
	Explanation string
}

/*
	实时加载的任务；目前暂不考虑支持
*/

type TaskHotLoad struct {
	// GetTaskExplanation() *TaskExplanation
	Name  string
	Paras string
	cmd   *exec.Cmd
}

func (t *TaskHotLoad) InstallTaskAndRun(name string, paramters string) error {

	var path = "madaoQT/task/"
	cmd := exec.Command("go", "install", path+name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		Logger.Errorf("Fail to install:%v", err)
		return errors.New(string(out))
	}

	cmd = exec.Command(name, "-config="+paramters)
	if cmd == nil {
		return errors.New("Fail to run task")
	}
	// Logger.Infof("Task Command:%v", cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// cmd.Stdin = os.Stdin
	cmd.Start()
	Logger.Infof("Task Command:%v, Task ID:%v", cmd.Args, cmd.Process.Pid)

	t.cmd = cmd
	return nil
}

func (t *TaskHotLoad) ExitTask() {
	if t.cmd == nil {
		Logger.Errorf("Invalid command to Exit")
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()
	Logger.Infof("Exiting task:%v", t.cmd.Process.Pid)
	select {
	case <-time.After(1 * time.Second):
		/*
			We would like to kill the process by signal, but there maybe some problem in windows; So we will use websocket to send the signal
		*/
		if err := t.cmd.Process.Kill(); err != nil {
			Logger.Errorf("Fail to kill task:%v", err)
		}
		Logger.Info("Succeed to kill task")
	case err := <-done:
		if err != nil {
			Logger.Errorf("Task exit with the error %v", err)
		}
	}
}

func ProcessTradeRoutine(exchange Exchange.IExchange,
	tradeConfig Exchange.TradeConfig,
	dbTrades *Mongo.Trades) chan TradeResult {

	// var balance interface{} // 实际余额后台返回为准
	// coin := Exchange.ParsePair(tradeConfig.Coin)[0]
	channel := make(chan TradeResult)
	stopTime := time.Now().Add(5 * time.Minute)

	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)

		// var dealAmount, totalCost, avePrice float64
		var dealAmount, totalCost, avePrice float64
		var trade *Exchange.TradeResult
		var depthInvalidCount int
		var errorCode TaskErrorType
		// var depth *Exchange.DepthValue
		var depth [][]Exchange.DepthPrice
		var tradePrice, tradeAmount float64
		var err error

		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("超出操作时间")
				errorCode = TaskErrorTimeout
				goto __ERROR
			}

			// 1. 根据深度情况计算价格下单
			// depth = exchange.GetDepthValue(tradeConfig.Pair,
			// 	tradeConfig.Price,
			// 	tradeConfig.Limit,
			// 	tradeConfig.Amount-dealAmount,
			// 	tradeConfig.Type)
			depth = exchange.GetDepthValue(tradeConfig.Pair)

			if depth == nil {
				Logger.Debugf("无操作价格")
				depthInvalidCount++
				/*
					连续十次无法达到操作价格，则退出
				*/
				if depthInvalidCount > 10 {
					errorCode = TaskInvalidDepth
					goto __ERROR
				}
				goto _NEXTLOOP
			} else {
				Logger.Debugf("深度:%v", depth)
				err, tradePrice, tradeAmount = getPlacedPrice(tradeConfig.Type,
					depth,
					tradeConfig.Price,
					tradeConfig.Limit,
					tradeConfig.Amount-dealAmount)

				if err != nil {
					Logger.Errorf("Trade Error:%v", err)
					Utils.SleepAsyncBySecond(3)
					goto _NEXTLOOP
				}

				Logger.Debugf("交易价格：%v 交易数量:%v", tradePrice, tradeAmount)
			}

			depthInvalidCount = 0

			trade = exchange.Trade(Exchange.TradeConfig{
				Pair:   tradeConfig.Pair,
				Type:   tradeConfig.Type,
				Amount: tradeConfig.Amount - dealAmount,
				Price:  tradePrice,
			})

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

			if trade != nil && trade.Error == nil {

				loop := 20

				for {
					Utils.SleepAsyncBySecond(3)

					info := exchange.GetOrderInfo(Exchange.OrderInfo{
						OrderID: trade.OrderID,
						Pair:    tradeConfig.Pair,
					})

					if info == nil || len(info) == 0 {
						Logger.Error("未取得订单信息")
						continue
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
						dbTrades.SetDone(trade.OrderID)
						goto __CheckDealAmount
					}

					loop--
					Logger.Debugf("等待成交...")

					if loop == 0 {
						Logger.Debugf("超时，取消订单...")
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
								Logger.Error("未取得订单信息")
								goto __ERROR
							}

							dbTrades.SetCanceled(trade.OrderID)

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
							Logger.Debugf("成功取消订单：%v, 已成交金额:%v", info[0].OrderID, dealAmount)
							goto __CheckDealAmount

						} else {
							Logger.Errorf("取消订单：%v失败，请手动操作", info[0].OrderID)
							errorCode = TaskUnableCancelOrder
							goto __ERROR
						}
					}
				}
			} else {
				Logger.Errorf("交易失败：%v", trade.Error)
				errorCode = TaskUnableTrade
				goto __ERROR
			}

		__ERROR:
			avePrice = totalCost / dealAmount
			channel <- TradeResult{
				Error:      errorCode,
				DealAmount: dealAmount,
				AvgPrice:   avePrice,
			}

			return
		__CheckBalance:
			// balance = exchange.GetBalance()[coin]
			avePrice = totalCost / dealAmount
			// Logger.Debugf("交易完成，余额：%v 成交均价：%v", balance, avePrice)
			channel <- TradeResult{
				Error: TaskErrorSuccess,
				// Balance:  balance.(map[string]interface{})["balance"].(float64),
				AvgPrice:   avePrice,
				DealAmount: dealAmount,
			}
			return

		__CheckDealAmount:
			Logger.Debugf("已成交:%v 总量:%v", dealAmount, tradeConfig.Amount)
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

func revertDepthArray(array []Exchange.DepthPrice) []Exchange.DepthPrice {
	var tmp Exchange.DepthPrice
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

// getPlacedPrice calcutes the price to be placed
func getPlacedPrice(tradeType Exchange.TradeType,
	items [][]Exchange.DepthPrice,
	price float64,
	limit float64,
	quantity float64) (error, float64, float64) {

	if items == nil || items[Exchange.DepthTypeAsks] == nil || items[Exchange.DepthTypeBids] == nil {
		return errors.New("Invalid depth list"), 0, 0
	}

	if len(items[Exchange.DepthTypeAsks]) == 0 || len(items[Exchange.DepthTypeBids]) == 0 {
		return errors.New("the lenth of the depth is 0"), 0, 0
	}

	var list []Exchange.DepthPrice
	var tradePrice, tradeQuantity float64

	if tradeType == Exchange.TradeTypeOpenLong || tradeType == Exchange.TradeTypeCloseShort || tradeType == Exchange.TradeTypeBuy {
		// we need the depth listed from low to high
		list = revertDepthArray(items[Exchange.DepthTypeAsks])
		limitPriceHigh := price * (1 + limit)
		Logger.Debugf("买入操作，接受最高价格：%v", limitPriceHigh)
		for _, item := range list {
			if item.Price <= limitPriceHigh {

				tradePrice = item.Price
				tradeQuantity += item.Quantity

				if tradeQuantity > quantity {
					tradeQuantity = quantity
					break
				}

			} else {
				Logger.Debugf("超出价格范围")
				return errors.New("exceed the price limit"), 0, 0
			}
		}

	} else {
		// we need the depth listed from higt to low
		list = items[Exchange.DepthTypeBids]

		limitPriceLow := price * (1 - limit)
		Logger.Debugf("卖出操作，接受最低价格：%v", limitPriceLow)
		for _, item := range list {
			if item.Price >= limitPriceLow {

				tradePrice = item.Price
				tradeQuantity += item.Quantity

				if tradeQuantity > quantity {
					tradeQuantity = quantity
					break
				}

			} else {
				Logger.Debugf("超出价格范围")
				return errors.New("exceed the price limit"), 0, 0
			}
		}
	}

	// limitPriceHigh := price * (1 + limit)
	// limitPriceLow := price * (1 - limit)
	// Logger.Debugf("有效价格范围：%v-%v", limitPriceLow, limitPriceHigh)

	// for _, item := range list {
	// 	if item.Price >= limitPriceLow && item.Price <= limitPriceHigh {

	// 		tradePrice = item.Price
	// 		tradeQuantity += item.Quantity

	// 		if tradeQuantity > quantity {
	// 			tradeQuantity = quantity
	// 			break
	// 		}

	// 	} else {
	// 		Logger.Debugf("超出价格范围")
	// 		break
	// 	}
	// }

	return nil, tradePrice, tradeQuantity
}

func TranslateToContractNumber(price float64, coinQuantity float64) int {
	return int(coinQuantity * price / 10)
}

func CalcDepthPrice(isFuture bool, exchange Exchange.IExchange, pair string, amount float64) (error, float64, float64) {

	var asks, bids []Exchange.DepthPrice
	var quantity, askPrice, bidPrice float64
	var askFlag, bidFlag bool
	depths := exchange.GetDepthValue(pair)
	Logger.Debugf("Future:%v 深度:%v", isFuture, depths)
	if depths != nil {
		asks = revertDepthArray(depths[Exchange.DepthTypeAsks])
		bids = depths[Exchange.DepthTypeBids]
	}

	amount *= 2

	if isFuture {
		quantity = amount / constContractRatio[Exchange.ParsePair(pair)[0]]
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
					break
				} else {
					totalQuantity += depth.Quantity
					totalAmount += depth.Quantity * depth.Price
				}
			}
		}
	}

	if askFlag && bidFlag {
		return nil, askPrice, bidPrice
	}

	return errors.New("Invalid depth"), 0, 0
}

/*
	静态加载的任务
*/

type StatusType int

const (
	StatusNone StatusType = iota
	StatusProcessing
	StatusOrdering
	StatusError
)

type ITask interface {
	GetTaskName() string
	GetDefaultConfig() interface{}
	GetBalances() map[string]interface{}
	GetTrades() []Mongo.TradesRecord

	Start(api string, secret string, configJSON string) error
	Close()
	GetStatus() int
}

func LoadStaticTask() []ITask {
	tasks := []ITask{}

	okexdiff := new(IAnalyzer)
	tasks = append(tasks, okexdiff)

	return tasks
}
