package exchange

type tickerValue struct{
	Tag string	// 用户调用者匹配
	Name string	// 用户交易所匹配
	Value interface{}
}

type tokenValues struct {
	High float64
	Low float64
	Open float64
	Close float64
	Last float64
	Time string
	Exchange string
}