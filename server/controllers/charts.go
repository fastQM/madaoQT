package controllers

import (
	"log"
	"time"

	"github.com/kataras/iris"

	Mongo "madaoQT/mongo"
	Task "madaoQT/task"
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
	fundManager := new(Task.OkexFundManage)
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

func (c *ChartsController) GetBalance() iris.Map {
	return iris.Map{
		"result": false,
	}
}
