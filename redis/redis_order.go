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
}

type RedisOrder struct {
	conn redis.Conn
}

func (r *RedisOrder) connect() error {

	conn, err := redis.Dial("tcp", "localhost:6379")
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

	_, err = r.conn.Do("lpush", key, string(value))
	if err != nil {
		return err
	}

	return nil

}
