package utils

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
)

const ModeEncrypt = 0
const ModeDecrypt = 1

type FileEncrypt struct {
	File   string
	Key    string
	Nonce  string
	Mode   int
	aesgcm cipher.AEAD
	_nonce []byte
}

func (f *FileEncrypt) init() {

	log.Printf("Filename:%s", f.File)
	if len(f.Key) > 32 || len(f.Key) == 0 {
		log.Fatalf("Invalide Key")
	}

	key := make([]byte, 32)
	copy(key, []byte(f.Key))
	log.Printf("Key:%x", key)

	if len(f.Nonce) > 12 || len(f.Nonce) == 0 {
		log.Fatalf("Invalide Nonce")
	}
	f._nonce = make([]byte, 12)
	copy(f._nonce, []byte(f.Nonce))
	log.Printf("Nonce:%x", f._nonce)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("Error:%v", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("Error:%v", err)
	}

	f.aesgcm = aesgcm
}

func (f *FileEncrypt) Encrypt() {
	f.execute(ModeEncrypt)
}

func (f *FileEncrypt) Decrypt() {
	f.execute(ModeDecrypt)
}

func (f *FileEncrypt) execute(mode int) {

	f.init()

	var PFunc func([]byte) []byte
	var suffix string
	var bufferSize int
	if mode == ModeEncrypt {
		PFunc = f.GCM_encrypt
		suffix = "-encrypted"
		bufferSize = 1024
	} else {
		PFunc = f.GCM_decrypt
		suffix = "-decrypted"
		bufferSize = 1040
	}

	inFile, err := os.Open(f.File)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	outFile, err := os.OpenFile(f.File+suffix, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	inFileReader := bufio.NewReader(inFile)
	outFileWriter := bufio.NewWriter(outFile)

	tmp := make([]byte, bufferSize)
	for {
		length, err := inFileReader.Read(tmp)
		log.Printf("Read len:%v error:%v", length, err)
		if err == io.EOF && length == 0 {
			outFileWriter.Flush()
			log.Printf("DONE!")
			return
		}

		plaintext := tmp[:length]
		ciphertext := PFunc(plaintext)
		_, err = outFileWriter.Write(ciphertext)
		// log.Printf("Write len:%v count:%v error:%v", len(ciphertext), count, err)
		if err != nil {
			log.Printf("Error:%v", err)
			return
		}

	}
}

func (f *FileEncrypt) GCM_encrypt(plaintext []byte) []byte {
	ciphertext := f.aesgcm.Seal(nil, f._nonce, plaintext, nil)
	// log.Printf("cipher:%x", ciphertext)
	return ciphertext
}

func (f *FileEncrypt) GCM_decrypt(ciphertext []byte) []byte {

	plaintext, err := f.aesgcm.Open(nil, f._nonce, ciphertext, nil)
	if err != nil {
		log.Fatalf("Error:%v", err)
	}
	// log.Printf("Plain:%x", plaintext)
	return plaintext
}

func ExampleNewGCM_encrypt() {
	// The key argument should be the AES key, either 16 or 32 bytes
	// to select AES-128 or AES-256.
	key := []byte("AES256Key-32Characters1234567890")
	plaintext := []byte("AES256Key-32Characters1234567890")

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	fmt.Printf("%x\n", ciphertext)
}

func ExampleNewGCM_decrypt() {
	// The key argument should be the AES key, either 16 or 32 bytes
	// to select AES-128 or AES-256.
	key := []byte("AES256Key-32Characters1234567890")
	ciphertext, _ := hex.DecodeString("1019aa66cd7c024f9efd0038899dae1973ee69427f5a6579eba292ffe1b5a260")

	nonce, _ := hex.DecodeString("37b8e8a308c354048d245f6d")

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("%s\n", plaintext)
	// Output: exampleplaintext
}
