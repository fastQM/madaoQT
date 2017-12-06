package utils

import (
	"log"
	"testing"
)

func TestRevert(t *testing.T) {
	array := []interface{}{
		"134", "345", "567",
	}

	array1 := []interface{}{
		"134", "345", "567", "9999",
	}

	array = RevertArray(array)
	log.Printf("array:%v", array)

	array1 = RevertArray(array1)
	log.Printf("array1:%v", array1)
}

func TestTimeLocation(t *testing.T) {
	time := FormatTime(1511680608304)
	log.Printf(time)
}

func TestGetRandomString(t *testing.T) {
	for i := 0; i < 10; i++ {
		string16 := GetRandomHexString(16)
		string32 := GetRandomHexString(32)
		log.Printf("string1: %s, string2: %s", string16, string32)
	}
}

// func TestCaseArray(t *testing.T) {
// 	channels := make([]chan string, 3)
// 	go func() {
// 		for {
// 			select {
// 			for _, channel := range channels{
// 			case msg := <-channel:
// 				return
// 			}
// 			}
// 		}
// 	}()

// }
