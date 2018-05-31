package utils

import (
	"math/rand"
	"os/exec"
	"runtime"
	"time"

	"github.com/kataras/golog"
)

const OS_Windows = "windows"
const OS_MacOS = "darwin"
const OS_Linux = "linux"
const OS_Unknown = "unknown"

/*
	初始化日志句柄
*/
var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetTimeFormat("2006-01-02 06:04:05")
}

func OpenBrowser(url string) {

	os := runtime.GOOS
	var cmd string

	Logger.Debugf("OS:%s", os)

	if os == OS_Windows {
		cmd = "explorer"
	} else if os == OS_MacOS {
		cmd = "open"
	} else if os == OS_Linux {
		cmd = "xdg-open"
	} else {
		//
		return
	}

	err := exec.Command(cmd, url).Start()
	if err != nil {
		Logger.Errorf("Fail to OpenBrowser:%v", err)
	}

}

func SleepAsyncBySecond(sec int64) {

	select {
	case <-time.After(time.Duration(sec) * time.Second):
		return
	}
}

func SleepAsyncByMillisecond(millisecond int64) {

	select {
	case <-time.After(time.Duration(millisecond) * time.Millisecond):
		return
	}
}

func RevertArray(array []interface{}) []interface{} {
	var tmp interface{}
	var length int

	if len(array)%2 != 0 {
		length = len(array) / 2
	} else {
		length = len(array)/2 - 1
	}
	for i := 0; i <= length; i++ {
		tmp = array[i]
		array[i] = array[len(array)-1-i]
		array[len(array)-1-i] = tmp

	}
	return array
}

func FormatTime(timestamp_ms int64) string {
	timeFormat := "2006-01-02 06:04:05"
	location, _ := time.LoadLocation("Asia/Shanghai")
	unixTime := time.Unix(timestamp_ms/1000, 0)
	return unixTime.In(location).Format(timeFormat)
}

func GetRandomHexString(length int) string {
	characters := []byte("abcdef0123456789")
	// rand.Seed(time.Now().UnixNano())
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
