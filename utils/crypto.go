package utils

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

const ModeEncrypt = 0
const ModeDecrypt = 1

const AESTypeFile = 0
const AESTypeBuffer = 1

type AESCrypto struct {
	Type     int
	FileName string
	Key      []byte
	Nonce    []byte
	Mode     int
	aesgcm   cipher.AEAD
	_nonce   []byte
}

func (f *AESCrypto) init() error {

	var err error

	if f.Type == AESTypeFile {
		if runtime.GOOS == "windows" {
			f.FileName = filepath.ToSlash(f.FileName)
		}
		log.Printf("[%s]Filename:%s", runtime.GOOS, f.FileName)
	}

	if len(f.Key) > 32 || len(f.Key) == 0 {

		fmt.Print("请输入密码:\r\n")
		f.Key, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
	}

	key := make([]byte, 32)
	copy(key, f.Key)
	// log.Printf("Key:%x", key)

	if len(f.Nonce) > 12 || len(f.Nonce) == 0 {

		fmt.Print("请输入随机数:\r\n")
		f.Nonce, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}

	}
	f._nonce = make([]byte, 12)
	copy(f._nonce, f.Nonce)
	// log.Printf("Nonce:%x", f._nonce)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	f.aesgcm = aesgcm

	return nil
}

func (f *AESCrypto) Encrypt() error {
	return f.execute(ModeEncrypt)
}

func (f *AESCrypto) Decrypt() error {
	return f.execute(ModeDecrypt)
}

func (f *AESCrypto) execute(mode int) error {

	if err := f.init(); err != nil {
		log.Printf("Init:%v", err)
		return err
	}

	var PFunc func([]byte) []byte
	var prefix string
	var bufferSize int

	filepath, filename := path.Split(f.FileName)
	// log.Printf("apth:%s", filepath)
	// log.Printf("name:%s", filename)

	if mode == ModeEncrypt {
		PFunc = f.GCM_encrypt
		prefix = "encrypted-"
		bufferSize = 4096
	} else {
		PFunc = f.GCM_decrypt
		prefix = "decrypted-"
		bufferSize = 4112
	}

	inFile, err := os.Open(f.FileName)
	if err != nil {
		return err
	}
	defer inFile.Close()

	log.Printf("Filename:%s", filepath+prefix+filename)

	outFile, err := os.OpenFile(filepath+prefix+filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
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
			return nil
		}

		plaintext := tmp[:length]
		ciphertext := PFunc(plaintext)
		_, err = outFileWriter.Write(ciphertext)
		// log.Printf("Write len:%v count:%v error:%v", len(ciphertext), count, err)
		if err != nil {
			return err
		}

	}
}

func (f *AESCrypto) EncryptInMemory(plain []byte) (error, []byte) {

	if err := f.init(); err != nil {
		log.Printf("Init:%v", err)
		return err, nil
	}

	var result []byte

	PFunc := f.GCM_encrypt
	bufferSize := 4112

	if f.Type == AESTypeFile {

		inFile, err := os.Open(f.FileName)
		if err != nil {
			return err, nil
		}
		defer inFile.Close()

		inFileReader := bufio.NewReader(inFile)

		tmp := make([]byte, bufferSize)
		for {
			length, err := inFileReader.Read(tmp)
			if err == io.EOF && length == 0 {
				return nil, result
			}

			plaintext := tmp[:length]
			ciphertext := PFunc(plaintext)
			result = append(result, ciphertext...)
		}
	} else {
		return nil, PFunc(plain)
	}

}

func (f *AESCrypto) DecryptInMemory(encrypted []byte) (error, []byte) {

	if err := f.init(); err != nil {
		log.Printf("Init:%v", err)
		return err, nil
	}

	var result []byte

	PFunc := f.GCM_decrypt
	bufferSize := 4112

	if f.Type == AESTypeFile {

		inFile, err := os.Open(f.FileName)
		if err != nil {
			return err, nil
		}
		defer inFile.Close()

		inFileReader := bufio.NewReader(inFile)

		tmp := make([]byte, bufferSize)
		for {
			length, err := inFileReader.Read(tmp)
			if err == io.EOF && length == 0 {
				return nil, result
			}

			plaintext := tmp[:length]
			ciphertext := PFunc(plaintext)
			result = append(result, ciphertext...)
		}
	} else {
		return nil, PFunc(encrypted)
	}

}

func (f *AESCrypto) GCM_encrypt(plaintext []byte) []byte {
	ciphertext := f.aesgcm.Seal(nil, f._nonce, plaintext, nil)
	// log.Printf("cipher:%x", ciphertext)
	return ciphertext
}

func (f *AESCrypto) GCM_decrypt(ciphertext []byte) []byte {

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
