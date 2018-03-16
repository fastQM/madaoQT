package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

var publicKey = []byte(`
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDDI/+oF9HeQj/KS4tKQ8PW8wFM
ErLXyewUc3ponJetgX5+vGlkTMHNLHtQQnA4uIeUqJ37nJM9rwKchw8BTB3qBofb
+nbAzfsiEDcGnIvA/MKl2leVyrxFfz0wCKZ2q3Pxwdlq8kHSNleLiuo6WRgFbJJp
L+lUkYI+b/LuAB8GtQIDAQAB
-----END PUBLIC KEY-----
`)

type RSA struct {
	privateKey []byte
}

func (p *RSA) LoadPrivateKey(key []byte) {
	p.privateKey = key
}

func (p *RSA) RsaEncrypt(origData []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("Fail to decode public key")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

func (p *RSA) RsaDecrypt(ciphertext []byte) ([]byte, error) {
	block, _ := pem.Decode(p.privateKey)
	if block == nil {
		return nil, errors.New("Fail to decode private key")

	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}
