package mongo

import (
	Global "madaoQT/config"

	"github.com/kataras/golog"
)

const MongoURL = "mongodb://localhost"
const MongoServer = "mongodb://34.218.78.117:28017"

// const MongoURL = "mongodb://192.168.0.102"
const Database = "madaoQT"

const ChartCollectin = "Chart"
const TradeRecordCollection = "TradeRecord"
const ExchangeCollection = "Exchanges"
const BalancesCollection = "Balances"
const FundCollection = "Funds"
const OkexDiffHistory = "OkexDiffHistory"

const ErrorNotConnected = "Mongo is not connected"

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
