package main

import (
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
		OpenTime:     time.Now(),
		Status:       MongoTrend.FundStatusOpen,
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
