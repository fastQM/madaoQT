package controllers

import (
	"log"

	"github.com/kataras/iris"

	Mongo "madaoQT/mongo"
)

type ChartsController struct {
	Ctx iris.Context
}

//
// GET: /charts

func (c *ChartsController) GetBy(coin string) iris.Map {
	okexdiff := &Mongo.OKExDiff{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExHistory",
		},
	}
	okexdiff.Connect()

	defer okexdiff.Close()

	Logger.Debugf("Coin:%v", coin)

	records, err := okexdiff.FindAll(map[string]interface{}{
		"coin": coin,
	})

	if err != nil {
		log.Printf("Error:%v", err)
		return iris.Map{
			"result": false,
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
