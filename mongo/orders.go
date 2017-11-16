package mongo



type OrderItem struct {
	Pair          string  `json:"pair"`
	Trigger       string  `json:"trigger"`
	SellLimitHigh float64 `json:"sellhigh"`
	SellLimitLow  float64 `json:"selllow"`
	BuyLimitHigh  float64 `json:"buyhigh"`
	BuyLimitLow   float64 `json:"buylow"`
	// priority
}