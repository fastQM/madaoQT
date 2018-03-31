package main

import (
	"log"
	Utils "madaoQT/utils"
)

func main() {
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
