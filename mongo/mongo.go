package mongo

const MongoURL = "mongodb://localhost"
const Database = "madaoQT"

const ChartCollectin = "Chart"
const TradeRecordCollection = "TradeRecord"
const ExchangeCollection = "Exchanges"
const OrderCollection = "Orders"

type DBConfig struct {
	CollectionName string
}
