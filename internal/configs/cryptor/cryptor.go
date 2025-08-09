package cryptor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// Cryptor wraps an RSA private and/or public key and provides
// methods for encryption and decryption.
type Cryptor struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// Opt defines a functional option type used to configure a Cryptor.
type Opt func(*Cryptor) error

// New creates a new Cryptor instance using the provided options.
func New(opts ...Opt) (*Cryptor, error) {
	c := &Cryptor{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithPrivateKeyPath loads an RSA private key from the specified PEM file
// and stores it in the Cryptor instance.
//
// The PEM file must contain a block with the type "RSA PRIVATE KEY".
func WithPrivateKeyPath(path string) Opt {
	return func(c *Cryptor) error {
		privData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		privBlock, _ := pem.Decode(privData)
		if privBlock == nil || privBlock.Type != "RSA PRIVATE KEY" {
			return errors.New("invalid private key format")
		}
		privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
		if err != nil {
			return err
		}
		c.privateKey = privKey
		return nil
	}
}

// WithPublicKeyPath loads an RSA public key from the specified PEM file
// and stores it in the Cryptor instance.
//
// The PEM file must contain a block with the type "PUBLIC KEY".
func WithPublicKeyPath(path string) Opt {
	return func(c *Cryptor) error {
		pubData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		pubBlock, _ := pem.Decode(pubData)
		if pubBlock == nil || pubBlock.Type != "PUBLIC KEY" {
			return errors.New("invalid public key format")
		}
		pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
		if err != nil {
			return err
		}
		rsaPubKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return errors.New("not an RSA public key")
		}
		c.publicKey = rsaPubKey
		return nil
	}
}

// Encrypt encrypts the given plaintext using the loaded public key.
// The function uses RSA PKCS#1 v1.5 padding.
func (c *Cryptor) Encrypt(data []byte) ([]byte, error) {
	if c.publicKey == nil {
		return nil, errors.New("public key is not loaded")
	}
	return rsa.EncryptPKCS1v15(rand.Reader, c.publicKey, data)
}

// Decrypt decrypts the given ciphertext using the loaded private key.
// The function uses RSA PKCS#1 v1.5 padding.
func (c *Cryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if c.privateKey == nil {
		return nil, errors.New("private key is not loaded")
	}
	return rsa.DecryptPKCS1v15(rand.Reader, c.privateKey, ciphertext)
}
