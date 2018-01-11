package task

import (
	"errors"
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
			CollectionName: "DiffOKExFutureFunds",
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

func (h *OkexFundManage) OpenPosition(exchangeType Exchange.ExchangeType,
	exchange Exchange.IExchange,
	batch string, coin string, opentype Exchange.TradeType) error {
	if exchangeType == Exchange.ExchangeTypeSpot {
		balances := exchange.GetBalance()
		coinBalance := balances[coin].(map[string]interface{})["balance"].(float64)
		usdtBalance := balances["usdt"].(map[string]interface{})["balance"].(float64)

		info := &Mongo.FundInfo{
			Type:        TypeSpot,
			Batch:       batch,
			Coin:        coin,
			OpenType:    Exchange.TradeTypeString[opentype],
			OpenTime:    time.Now(),
			OpenBalance: coinBalance,
			Exchange:    exchange.GetExchangeName(),
			Status:      Mongo.FundStatusOpen,
		}

		if err := h.fundDB.Insert(info); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		info = &Mongo.FundInfo{
			Type:        TypeSpot,
			Batch:       batch,
			Coin:        "usdt",
			OpenType:    Exchange.TradeTypeString[Exchange.RevertTradeType(opentype)],
			OpenTime:    time.Now(),
			OpenBalance: usdtBalance,
			Exchange:    exchange.GetExchangeName(),
			Status:      Mongo.FundStatusOpen,
		}

		if err := h.fundDB.Insert(info); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		return nil

	} else if exchangeType == Exchange.ExchangeTypeFuture {
		balances := exchange.GetBalance()
		balance := balances[coin].(map[string]interface{})["balance"].(float64)
		bond := balances[coin].(map[string]interface{})["bond"].(float64)

		info := &Mongo.FundInfo{
			Type:        TypeFuture,
			Batch:       batch,
			Coin:        coin,
			OpenType:    Exchange.TradeTypeString[opentype],
			OpenTime:    time.Now(),
			OpenBalance: balance,
			Exchange:    exchange.GetExchangeName(),
			Status:      Mongo.FundStatusOpen,
		}

		if err := h.fundDB.Insert(info); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		info = &Mongo.FundInfo{
			Type:        TypeFuture,
			Batch:       batch,
			Coin:        "bond",
			OpenType:    Exchange.TradeTypeString[Exchange.RevertTradeType(opentype)],
			OpenTime:    time.Now(),
			OpenBalance: bond,
			Exchange:    exchange.GetExchangeName(),
			Status:      Mongo.FundStatusOpen,
		}

		if err := h.fundDB.Insert(info); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		return nil
	}

	return errors.New("invalid type")
}

func (h *OkexFundManage) ClosePosition(exchangeType Exchange.ExchangeType,
	exchange Exchange.IExchange,
	batch string, coin string) error {

	now := time.Now()

	if exchangeType == Exchange.ExchangeTypeSpot {
		balances := exchange.GetBalance()
		coinBalance := balances[coin].(map[string]interface{})["balance"].(float64)
		usdtBalance := balances["usdt"].(map[string]interface{})["balance"].(float64)

		if err := h.fundDB.Update(map[string]interface{}{
			"type":  TypeSpot,
			"batch": batch,
			"coin":  coin,
		}, map[string]interface{}{
			"closetime":    now,
			"closebalance": coinBalance,
			"status":       Mongo.FundStatusClosed,
		}); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		if err := h.fundDB.Update(map[string]interface{}{
			"type":  TypeSpot,
			"batch": batch,
			"coin":  "usdt",
		}, map[string]interface{}{
			"closetime":    now,
			"closebalance": usdtBalance,
			"status":       Mongo.FundStatusClosed,
		}); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}
	} else if exchangeType == Exchange.ExchangeTypeFuture {
		balances := exchange.GetBalance()
		balance := balances[coin].(map[string]interface{})["balance"].(float64)
		bond := balances[coin].(map[string]interface{})["bond"].(float64)

		if err := h.fundDB.Update(map[string]interface{}{
			"type":  TypeFuture,
			"batch": batch,
			"coin":  coin,
		}, map[string]interface{}{
			"closetime":    now,
			"closebalance": balance,
			"status":       Mongo.FundStatusClosed,
		}); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}

		if err := h.fundDB.Update(map[string]interface{}{
			"type":  TypeFuture,
			"batch": batch,
			"coin":  "bond",
		}, map[string]interface{}{
			"closetime":    now,
			"closebalance": bond,
			"status":       Mongo.FundStatusClosed,
		}); err != nil {
			Logger.Errorf("Error:%v", err)
			return err
		}
	}

	return errors.New("Invalid exchange type")
}

func (h *OkexFundManage) changePositionStatus(batch string, coin string) {

}

func (h *OkexFundManage) CalcRatio() {

	err, records := h.fundDB.FindAll()
	if err != nil {
		return
	}

	for _, record := range records {
		if record.Status == Mongo.FundStatusClosed {
			Logger.Infof("Batch:%s", record.Batch)
			if record.OpenType == Exchange.TradeTypeString[Exchange.TradeTypeBuy] || record.OpenType == Exchange.TradeTypeString[Exchange.TradeTypeOpenLong] {
				Logger.Infof("Ratio:%v", (record.CloseBalance-record.OpenBalance)*100/record.OpenBalance)
			} else {
				Logger.Infof("Ratio:%v", (record.OpenBalance-record.CloseBalance)*100/record.OpenBalance)
			}
		}
	}
}
