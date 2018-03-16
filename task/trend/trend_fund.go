package trend

import (
	"log"
	"time"

	Exchange "madaoQT/exchange"
	MongoTrend "madaoQT/mongo/trend"
)

type FundManager struct {
	funds MongoTrend.Funds
}

func (p *FundManager) Init(collection MongoTrend.Funds) {
	p.funds = collection
}

func (p *FundManager) OpenPosition(batch string,
	timestamp int64,
	pair string,
	futureType Exchange.TradeType,
	futureOpen float64,
	futureAmount float64) error {

	info := &MongoTrend.FundInfo{
		Batch:        batch,
		Pair:         pair,
		FutureType:   Exchange.TradeTypeString[futureType],
		FutureOpen:   futureOpen,
		FutureAmount: futureAmount,
		// OpenTime:     time.Unix(timestamp, 0),
		OpenTime: time.Now(),
		Status:   MongoTrend.FundStatusOpen,
	}

	if err := p.funds.Insert(info); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (p *FundManager) ClosePosition(batch string, futureClose float64, result string) error {

	if err := p.funds.Update(map[string]interface{}{
		"batch": batch,
	}, map[string]interface{}{
		"futureclose": futureClose,
		"closetime":   time.Now(),
		"status":      result,
	}); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (p *FundManager) CheckPosition() (err error, records []MongoTrend.FundInfo) {
	if err, records = p.funds.Find(map[string]interface{}{
		"status": MongoTrend.FundStatusOpen,
	}); err != nil {
		return err, nil
	}

	return nil, records
}

func (p *FundManager) GetFailedPositions() (err error, records []MongoTrend.FundInfo) {

	if err, records = p.funds.Find(map[string]interface{}{
		"status": MongoTrend.FundStatusError,
	}); err != nil {
		return err, nil
	}
	return nil, records

}

func (p *FundManager) FixFailedPosition(updates map[string]interface{}) error {

	updates["closetime"] = time.Now()
	updates["status"] = MongoTrend.FundStatusClose

	if err := p.funds.Update(map[string]interface{}{
		"batch": updates["batch"],
	}, updates); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	return nil
}

func (p *FundManager) CheckDailyProfit(date time.Time) (error, float64) {
	if err, records := p.funds.GetDailySuccessRecords(date); err != nil {
		return err, 0
	} else {
		return nil, p.CheckProfit(records)
	}
}

func (p *FundManager) CheckProfit(records []MongoTrend.FundInfo) float64 {

	var total float64

	for _, record := range records {
		coin := Exchange.ParsePair(record.Pair)[0]
		var futureProfit, fee float64

		amount := constContractRatio[coin] * record.FutureAmount
		if record.FutureOpen == 0 || record.FutureClose == 0 {
			log.Printf("[%v]无效合约数据", record.Batch)
		} else {
			if record.FutureType == Exchange.TradeTypeString[Exchange.TradeTypeOpenLong] {
				futureProfit = (amount/record.FutureClose - amount/record.FutureOpen) * (-1)
			} else {
				futureProfit = (amount/record.FutureOpen - amount/record.FutureClose) * (-1)
			}

			fee += (amount/record.FutureOpen + amount/record.FutureClose) * 0.0005
		}

		// log.Printf("[%v][%v]收益:%.8f 手续费:%.8f 净收益:%.8f", record.Batch, record.CloseTime, futureProfit, fee, (futureProfit - fee))
		total += (futureProfit - fee)
	}

	log.Printf("总收益:%.8f", total)
	return total
}
