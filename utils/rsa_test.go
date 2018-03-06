package utils

import (
	"log"
	"testing"
)

func TestRSA(t *testing.T) {
	encrypted, err := RsaEncrypt([]byte("it is a test"))
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	decrypted, err := RsaDecrypt(encrypted)
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	log.Printf("Plain:%s", decrypted)
}
