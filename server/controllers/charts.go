package controllers

import (
	"log"
	"time"

	"github.com/kataras/iris"

	"madaoQT/exchange"
	Mongo "madaoQT/mongo"
	MongoTrend "madaoQT/mongo/trend"
	Task "madaoQT/task"
	OkexDiff "madaoQT/task/okexdiff"
	Trend "madaoQT/task/trend"
)

type ChartsController struct {
	Ctx iris.Context
}

//
// GET: /charts

func (c *ChartsController) GetDiffBy(coin string) iris.Map {
	okexdiff := &Mongo.OKExDiff{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExHistory",
		},
	}
	okexdiff.Connect()

	defer okexdiff.Close()

	now := time.Now()
	start := now.Add(-12 * time.Hour)
	log.Printf("start:%v stop:%v", start, now)
	records, err := okexdiff.FindAll("eth", start, now)

	if err != nil {
		return iris.Map{
			"result": false,
			"error":  err.Error(),
		}
	}

	log.Printf("Count:%v", len(records))

	length := len(records)
	futures := make([]float64, length)
	spots := make([]float64, length)
	diffs := make([]float64, length)
	times := make([]string, length)
	spotVolumns := make([]float64, length)
	futureVolumns := make([]float64, length)
	for i, record := range records {
		futures[i] = record.FuturePrice
		spots[i] = record.SpotPrice
		diffs[i] = record.Diff
		times[i] = record.Time.Format("2006-01-02 15:04:05")
		spotVolumns[i] = record.SpotVolume * record.SpotPrice
		futureVolumns[i] = record.FutureVolume * 10
	}

	return iris.Map{
		"result": true,
		"data": map[string]interface{}{
			"futures":       futures,
			"futurevolumns": futureVolumns,
			"spots":         spots,
			"spotvolumns":   spotVolumns,
			"diffs":         diffs,
			"times":         times,
		},
	}
}

func (c *ChartsController) GetProfit() iris.Map {
	fundManager := new(OkexDiff.OkexFundManage)
	fundManager.Init()
	date := time.Date(2018, 2, 1, 0, 0, 0, 0, time.Local)
	today := time.Now()

	days := int(today.Sub(date)/(24*time.Hour)) + 1

	log.Printf("days:%v", days)
	var index int
	timeList := make([]string, days)
	profitList := make([]float64, days)

	for {
		var profit float64
		var err error
		if err, profit = fundManager.CheckDailyProfit(date); err != nil {
			return iris.Map{
				"result": false,
				"error":  err.Error(),
			}
		}

		timeList[index] = date.Format("2006-01-02")
		profitList[index] = profit
		index++

		if date.AddDate(0, 0, 1).After(today) {
			break
		} else {
			date = date.AddDate(0, 0, 1)
		}
	}

	return iris.Map{
		"result": true,
		"data": map[string]interface{}{
			"times":   timeList,
			"profits": profitList,
		},
	}
}

func (c *ChartsController) GetExamples() iris.Map {

	var result []exchange.KlineValue

	filename := "poloniex-15min"

	result = exchange.LoadHistory(filename)

	areas := exchange.StrategyTrendTest(result, true, true)

	return iris.Map{
		"result": true,
		"data":   areas,
	}
}

func (c *ChartsController) GetProfitBy(name string) iris.Map {

	var collection string
	if name == "binance" {
		collection = Task.TrendBalanceBinance
	} else if name == "okex" {
		collection = Task.TrendBalanceOKEX
	} else {
		return iris.Map{
			"result": false,
			"error":  errorMessage[errorCodeInvalidParameters],
		}
	}

	db := &MongoTrend.TrendMongo{
		BalanceCollectionName: collection,
		Server:                Trend.MongoServer,
		Sock5Proxy:            "SOCKS5:127.0.0.1:1080",
	}
	if err := db.Connect(); err != nil {
		Logger.Errorf("Error3:%v", err)
		return iris.Map{
			"result": false,
			"error":  errorMessage[errorCodeMongoDisconnect],
		}
	}

	balanceManager := new(Trend.BalanceManager)
	balanceManager.Init(&db.BalanceCollection)

	date := time.Date(2018, 4, 1, 0, 0, 0, 0, time.Local)
	today := time.Now()

	type Item struct {
		Coin    string
		Balance float64
	}

	type Balance struct {
		Time  time.Time
		Coins []Item
	}

	var records []Balance

	for {
		err, last := balanceManager.GetDialyLast(date)
		if err == nil {
			item := Balance{
				Time: date,
			}

			item.Coins = make([]Item, len((*last).Item))
			for i, coin := range (*last).Item {
				item.Coins[i].Coin = coin.Coin
				item.Coins[i].Balance = coin.Balance
			}

			records = append(records, item)
		}

		if date.AddDate(0, 0, 1).After(today) {
			break
		} else {
			date = date.AddDate(0, 0, 1)
		}
	}

	return iris.Map{
		"result": true,
		"data":   records,
	}
}
