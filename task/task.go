package task

import (
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/kataras/golog"

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
	Logger.SetTimeFormat("2006-01-02 06:04:05")
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
	stopTime := time.Now().Add(1 * time.Minute)

	Logger.Debugf("Trade Params:%v", tradeConfig)

	go func() {
		defer close(channel)

		// var dealAmount, totalCost, avePrice float64
		var dealAmount float64
		var trade *Exchange.TradeResult
		var depthInvalidCount int
		var errorCode TaskErrorType
		var depth *Exchange.DepthValue

		for {

			if time.Now().After(stopTime) {
				Logger.Debugf("超出操作时间")
				errorCode = TaskErrorTimeout
				goto __ERROR
			}

			// 1. 根据深度情况计算价格下单
			depth = exchange.GetDepthValue(tradeConfig.Pair,
				tradeConfig.Price,
				tradeConfig.Limit,
				tradeConfig.Amount-dealAmount,
				tradeConfig.Type)

			Logger.Debugf("深度信息:%v 下单信息：%v 已成交额度：%v", depth, tradeConfig, dealAmount)

			if depth == nil || depth.LimitTradeAmount == 0 || depth.LimitTradePrice == 0 {
				Logger.Debugf("无操作价格:%v", depth)
				depthInvalidCount++
				/*
					连续十次无法达到操作价格，则退出
				*/
				if depthInvalidCount > 10 {
					errorCode = TaskInvalidDepth
					goto __ERROR
				}
				goto _NEXTLOOP
			}

			depthInvalidCount = 0

			trade = exchange.Trade(Exchange.TradeConfig{
				Pair:   tradeConfig.Pair,
				Type:   tradeConfig.Type,
				Amount: tradeConfig.Amount - dealAmount,
				Price:  depth.LimitTradePrice,
			})

			if err := dbTrades.Insert(&Mongo.TradesRecord{
				Batch:    tradeConfig.Batch,
				Oper:     Exchange.TradeTypeString[tradeConfig.Type],
				Exchange: exchange.GetExchangeName(),
				Pair:     tradeConfig.Pair,
				Quantity: depth.LimitTradeAmount,
				Price:    depth.LimitTradePrice,
				OrderID:  trade.OrderID,
			}); err != nil {
				Logger.Errorf("保存交易操作失败:%v", err)
			}

			if trade != nil && trade.Error == nil {

				loop := 10

				for {
					Utils.SleepAsyncBySecond(1)

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
						// totalCost += (info[0].AvgPrice * info[0].DealAmount) //手续费如何？
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
							// totalCost += (info[0].AvgPrice * info[0].DealAmount)
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
			channel <- TradeResult{
				Error:      errorCode,
				DealAmount: dealAmount,
			}

			return
		__CheckBalance:
			// balance = exchange.GetBalance()[coin]
			// avePrice = totalCost / dealAmount
			// Logger.Debugf("交易完成，余额：%v 成交均价：%v", balance, avePrice)
			channel <- TradeResult{
				Error: TaskErrorSuccess,
				// Balance:  balance.(map[string]interface{})["balance"].(float64),
				// AvgPrice: avePrice,
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

/*
	静态加载的任务
*/

type ITask interface {
	GetTaskName() string
	GetDefaultConfig() interface{}
	GetBalances() map[string]interface{}
	GetTrades() []Mongo.TradesRecord
	Start(api string, secret string, configJSON string) error
	Close()
}

func LoadStaticTask() []ITask {
	tasks := []ITask{}

	okexdiff := new(IAnalyzer)
	tasks = append(tasks, okexdiff)

	return tasks
}
