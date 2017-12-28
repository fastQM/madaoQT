package fund

import (
	Mongo "madaoQT/mongo"
	"time"

	"github.com/kataras/golog"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
	Logger.SetPrefix("[FUND]")
}

type FundManage struct {
	fundDB *Mongo.Funds
}

func (f *FundManage) Init() {
	f.fundDB = &Mongo.Funds{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFutureFunds",
		},
	}

	if err := f.fundDB.Connect(); err != nil {
		Logger.Errorf("ERR:%v", err)
		return
	}

}

func (f *FundManage) Close() {
	if f.fundDB != nil {
		f.fundDB.Close()
	}
}

func (f *FundManage) SaveBalanceBeforeOpen(index string, exchange string, balance float64) {
	info := &Mongo.FundInfo{
		Batch:       index,
		OpenTime:    time.Now(),
		OpenBalance: balance,
		Exchange:    exchange,
		Status:      Mongo.FundStatusOpen,
	}

	if err := f.fundDB.Insert(info); err != nil {
		Logger.Errorf("Error:%v", err)
		return
	}
}

func (f *FundManage) SaveBalanceAfterClose(index string, balance float64) {
	if err := f.fundDB.Update(map[string]interface{}{
		"batch": index,
	}, map[string]interface{}{
		"closetime":    time.Now(),
		"closebalance": balance,
	}); err != nil {
		Logger.Errorf("Error:%v", err)
		return
	}
}
