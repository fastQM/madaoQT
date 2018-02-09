package mongo

import (
	Global "madaoQT/config"

	"github.com/kataras/golog"
)

const MongoURL = "mongodb://localhost"

// const MongoURL = "mongodb://192.168.0.102"
const Database = "madaoQT"

const ChartCollectin = "Chart"
const TradeRecordCollection = "TradeRecord"
const ExchangeCollection = "Exchanges"
const BalancesCollection = "Balances"
const FundCollection = "Funds"
const OkexDiffHistory = "OkexDiffHistory"

type DBConfig struct {
	CollectionName string
}

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[MONG]")
}
