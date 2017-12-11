package mongo

const MongoURL = "mongodb://localhost"
const Database = "madaoQT"

const ChartCollectin = "Chart"
const TradeRecordCollection = "TradeRecord"
const ExchangeCollection = "Exchanges"

type DBConfig struct {
	CollectionName string
}
