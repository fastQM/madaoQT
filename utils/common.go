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