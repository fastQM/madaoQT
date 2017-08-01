package redis

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/garyburd/redigo/redis"
)

type OrderItem struct {
	Pair          string  `json:"pair"`
	Trigger       string  `json:"trigger"`
	SellLimitHigh float64 `json:"sellhigh"`
	SellLimitLow  float64 `json:"selllow"`
	BuyLimitHigh  float64 `json:"buyhigh"`
	BuyLimitLow   float64 `json:"buylow"`
	// priority
}

type RedisOrder struct {
	conn redis.Conn
}

func (r *RedisOrder) connect() error {

	conn, err := redis.Dial("tcp", RedisServer)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	r.conn = conn
	return nil
}

func (r *RedisOrder) SaveOrder(exchange string, order OrderItem) error {

	err := r.connect()

	if err != nil {
		return errors.New("Redis is not connected")
	}

	defer r.conn.Close()

	key := "orders-" + exchange

	value, err := json.Marshal(order)
	if err != nil {
		return err
	}

	_, err = r.conn.Do("sadd", key, string(value))
	if err != nil {
		return err
	}

	return nil

}

func (r *RedisOrder) LoadOrders(exchange string) ([]OrderItem, error) {

	err := r.connect()

	if err != nil {
		return nil, errors.New("Redis is not connected")
	}

	defer r.conn.Close()

	key := "orders-" + exchange

	tmp, err := r.conn.Do("smembers", key)
	if err != nil {
		return nil, err
	}

	charts := tmp.([]interface{})
	if charts == nil || len(charts) == 0 {
		err = errors.New("No data found")
		return nil, err
	}

	orders := make([]OrderItem, len(charts))

	for i := 0; i < len(charts); i++ {
		value, err := redis.String(charts[i], nil)
		// fmt.Printf("chart:%v\r\n", value)
		if err != nil || value == "" {
			return nil, err
		}

		err = json.Unmarshal([]byte(value), &orders[i])
		if err != nil {
			fmt.Printf("err:%v", err)
			return nil, err
		}

	}

	return orders, nil

}
