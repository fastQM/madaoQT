package utils

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {

	file := "test.mp4"
	encrypt := AESCrypto{
		FileName: file,
	}

	encrypt.Encrypt()

	encrypt = AESCrypto{
		FileName: file + "-encrypted",
	}

	encrypt.Decrypt()
}
