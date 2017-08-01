package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	redis "github.com/garyburd/redigo/redis"
)

const RedisServer = "localhost:6379"

type ChartsHistory struct {
	conn redis.Conn

	Charts []ChartItem
}

type ChartItem struct {
	Date            int64   `json:"date"`
	Hm              string  `json:"hm"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
}

func (t *ChartsHistory) connect() error {
	conn, err := redis.Dial("tcp", RedisServer)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	t.conn = conn
	return nil
}

func (t *ChartsHistory) LoadCharts(name string, period int) error {

	err := t.connect()

	if err != nil {
		return errors.New("Redis is not connected")
	}

	defer t.conn.Close()

	key := "charts-poloniex-" + name
	log.Printf("Load datas from:%v", key)
	tmp, err := t.conn.Do("lrange", key, 0, -1)
	if err != nil {
		fmt.Printf("fail to load charts:%v", err)
		return err
	}

	charts := tmp.([]interface{})
	if charts == nil || len(charts) == 0 {
		err = errors.New("No data found")
		return err
	}

	t.Charts = make([]ChartItem, len(charts))

	for i := 0; i < len(charts); i++ {
		value, err := redis.String(charts[i], nil)
		// fmt.Printf("chart:%v\r\n", value)
		if err != nil || value == "" {
			return err
		}

		err = json.Unmarshal([]byte(value), &t.Charts[i])
		if err != nil {
			fmt.Printf("err:%v", err)
			return err
		}

	}

	// for i := 0; i < len(t.Charts); i++ {
	// 	fmt.Printf("%v.chart:%v\r\n", i, t.Charts[i])
	// }

	return nil
}
