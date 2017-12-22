package utils

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"errors"
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
		bufferSize = 4096
	} else {
		PFunc = f.GCM_decrypt
		suffix = "-decrypted"
		bufferSize = 4112
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
		// log.Printf("Read len:%v error:%v", length, err)
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

func GCM_encrypt(password string, username string, plaintext string) (error, []byte) {

	if len(password) > 32 || len(password) == 0 {
		return errors.New("Invalide password"), nil
	}

	key := make([]byte, 32)
	copy(key, []byte(password))

	if len(username) > 12 || len(username) == 0 {
		return errors.New("Invalide username"), nil
	}
	nonce := make([]byte, 12)
	copy(nonce, []byte(username))

	block, err := aes.NewCipher(key)
	if err != nil {
		return err, nil
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err, nil
	}

	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)
	return nil, ciphertext
}

func GCM_decrypt(password string, username string, ciphertext string) (error, []byte) {

	if len(password) > 32 || len(password) == 0 {
		return errors.New("Invalide password"), nil
	}

	key := make([]byte, 32)
	copy(key, []byte(password))

	if len(username) > 12 || len(username) == 0 {
		return errors.New("Invalide username"), nil
	}
	nonce := make([]byte, 12)
	copy(nonce, []byte(username))

	block, err := aes.NewCipher(key)
	if err != nil {
		return err, nil
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err, nil
	}

	plaintext, err := aesgcm.Open(nil, nonce, []byte(ciphertext), nil)
	if err != nil {
		return err, nil
	}

	return nil, plaintext
}
