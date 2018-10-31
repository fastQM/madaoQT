package main

import (
	"log"
	Utils "madaoQT/utils"
	"os"
	"runtime/debug"
	"time"
)

func main1() {
	// file := "E:\\Backup\\data\\rsa.txt"

	// encrypt := utils.FileEncrypt{
	// 	File: file,
	// }

	// if err := encrypt.Encrypt(); err != nil {
	// 	log.Printf("error:%v", err)
	// }

	crypto := Utils.AESCrypto{
		Type: Utils.AESTypeBuffer,
	}

	var encrypted []byte
	if err, result := crypto.EncryptInMemory([]byte("hello,world")); err != nil {
		log.Printf("error:%v", err)
	} else {
		log.Printf("Encrypted:%x", result)
		encrypted = result
	}

	if err, result := crypto.DecryptInMemory(encrypted); err != nil {
		log.Printf("error:%v", err)
	} else {
		log.Printf("Plain:%s", string(result))
	}

}

func TestFunction(divider float64) {
	var MakecoreData *int = nil
	*MakecoreData = 10000
	log.Printf("I am here:%v", MakecoreData)
}

func main() {

	divider := 0.0
	for {
		select {
		case <-time.After(1 * time.Second):
			go func() {
				defer func() {
					if err := recover(); err != nil {
						result := string(debug.Stack())
						log.Printf("[EXCEPTION]%v", result)
					}
				}()
				if divider < 3 {
					divider++
				} else {
					os.Exit(0)
				}
				TestFunction(divider)

			}()

		}
	}

}
