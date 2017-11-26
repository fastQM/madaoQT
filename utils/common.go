package utils

import (
	"time"
)

func SleepAsyncBySecond(sec time.Duration){

	select{
	case <- time.After(sec*time.Second):
			return;
	}
}

func RevertArray(array ...interface{}) []interface{} {
	var tmp interface{}
	var length int

	if len(array)%2 != 0 {
		length = len(array)/2
	} else {
		length = len(array)/2-1
	}
	for i:=0;i<=length;i++{
		tmp = array[i]
		array[i] = array[len(array)-1-i]
		array[len(array)-1-i] = tmp

	}
	return array
}

func FormatTime(timestamp_ms int64) string {
	timeFormat := "2006-01-02 06:04:05"
	location,_ := time.LoadLocation("Asia/Shanghai")
	unixTime := time.Unix(timestamp_ms/1000, 0)
	return unixTime.In(location).Format(timeFormat)
}