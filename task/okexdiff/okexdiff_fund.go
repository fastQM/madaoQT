package okexdiff

import (
	"errors"
	"log"
	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	Task "madaoQT/task"
	"time"
)

type OkexFundManage struct {
	fundDB *Mongo.Funds
}

const TypeSpot = "spot"
const TypeFuture = "future"

func (h *OkexFundManage) Init() error {
	h.fundDB = &Mongo.Funds{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFunds",
		},
	}

	if err := h.fundDB.Connect(); err != nil {
		Logger.Errorf("ERR:%v", err)
		return err
	}

	return nil
}

func (h *OkexFundManage) Close() {
	if h.fundDB != nil {
		h.fundDB.Close()
	}
}

func (h *OkexFundManage) Refresh() {
	if h.fundDB != nil {
		h.fundDB.Refresh()
	}
}

func (h *OkexFundManage) OpenPosition(batch string,
	pair string,
	spotType Exchange.TradeType,
	futureType Exchange.TradeType,
	spotOpen float64,
	spotAmount float64,
	futureOpen float64,
	futureAmount float64) error {

	info := &Mongo.FundInfo{
		Batch:        batch,
		Pair:         pair,
		SpotType:     Exchange.TradeTypeString[spotType],
		FutureType:   Exchange.TradeTypeString[futureType],
		SpotOpen:     spotOpen,
		SpotAmount:   spotAmount,
		FutureOpen:   futureOpen,
		FutureAmount: futureAmount,
		OpenTime:     time.Now(),
		Status:       Mongo.FundStatusOpen,
	}

	if err := h.fundDB.Insert(info); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (h *OkexFundManage) ClosePosition(batch string, spotClose float64, futureClose float64, result string) error {

	// var result string
	// if success {
	// 	result = Mongo.FundStatusClosed
	// } else {
	// 	result = Mongo.FUndStatusError
	// }

	if err := h.fundDB.Update(map[string]interface{}{
		"batch": batch,
	}, map[string]interface{}{
		"spotclose":   spotClose,
		"futureclose": futureClose,
		"closetime":   time.Now(),
		"status":      result,
	}); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (h *OkexFundManage) CheckPosition() (err error, records []Mongo.FundInfo) {
	if err, records = h.fundDB.Find(map[string]interface{}{
		"status": Mongo.FundStatusOpen,
	}); err != nil {
		return err, nil
	}

	return nil, records
}

func (h *OkexFundManage) GetFailedPositions() (err error, records []Mongo.FundInfo) {
	if h.fundDB != nil {
		if err, records = h.fundDB.Find(map[string]interface{}{
			"status": Mongo.FundStatusError,
		}); err != nil {
			return err, nil
		}
		return nil, records
	}

	return errors.New(Task.TaskErrorMsg[Task.TaskLostMongodb]), nil
}

func (h *OkexFundManage) FixFailedPosition(updates map[string]interface{}) error {

	updates["closetime"] = time.Now()
	updates["status"] = Mongo.FundStatusClose

	if err := h.fundDB.Update(map[string]interface{}{
		"batch": updates["batch"],
	}, updates); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (h *OkexFundManage) CheckDailyProfit(date time.Time) (error, float64) {
	if err, records := h.fundDB.GetDailySuccessRecords(date); err != nil {
		return err, 0
	} else {
		return nil, h.CheckProfit(records)
	}
}

func (h *OkexFundManage) CheckProfit(records []Mongo.FundInfo) float64 {

	var total float64
	// var records []Mongo.FundInfo
	// if err, records = h.fundDB.Find(map[string]interface{}{
	// 	"status": Mongo.FundStatusClose,
	// }); err != nil {
	// 	log.Printf("Error:%v", err)
	// 	return
	// }

	for _, record := range records {
		coin := Exchange.ParsePair(record.Pair)[0]
		var spotProfit, futureProfit, fee float64

		if record.SpotType == Exchange.TradeTypeString[Exchange.TradeTypeBuy] {
			spotProfit = (record.SpotClose - record.SpotOpen) * record.SpotAmount
		} else {
			spotProfit = (record.SpotOpen - record.SpotClose) * record.SpotAmount
		}
		fee = (record.SpotOpen + record.SpotClose) * record.SpotAmount * 0.002

		amount := constContractRatio[coin] * record.FutureAmount
		if record.FutureOpen == 0 || record.FutureClose == 0 {
			log.Printf("[%v]无效合约数据", record.Batch)
		} else {
			if record.FutureType == Exchange.TradeTypeString[Exchange.TradeTypeOpenLong] {
				futureProfit = (amount/record.FutureClose - amount/record.FutureOpen) * record.FutureClose * (-1)
			} else {
				futureProfit = (amount/record.FutureOpen - amount/record.FutureClose) * record.FutureClose * (-1)
			}

			fee += 100 * 2 * 0.0005
		}

		log.Printf("[%v][%v]现货收益:%v 合约收益:%v 收益:%v", record.Batch, record.CloseTime, spotProfit, futureProfit, (spotProfit + futureProfit - fee))
		total += (spotProfit + futureProfit - fee)
	}

	log.Printf("总收益:%v", total)
	return total
}
