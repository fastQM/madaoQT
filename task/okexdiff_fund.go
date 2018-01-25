package task

import (
	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
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

// func (h *OkexFundManage) CalcRatio() {

// 	err, records := h.fundDB.FindAll()
// 	if err != nil {
// 		return
// 	}

// 	for _, record := range records {
// 		if record.Status == Mongo.FundStatusClosed {
// 			Logger.Infof("Batch:%s", record.Batch)
// 			if record.OpenType == Exchange.TradeTypeString[Exchange.TradeTypeBuy] || record.OpenType == Exchange.TradeTypeString[Exchange.TradeTypeOpenLong] {
// 				Logger.Infof("Ratio:%v", (record.CloseBalance-record.OpenBalance)*100/record.OpenBalance)
// 			} else {
// 				Logger.Infof("Ratio:%v", (record.OpenBalance-record.CloseBalance)*100/record.OpenBalance)
// 			}
// 		}
// 	}
// }
