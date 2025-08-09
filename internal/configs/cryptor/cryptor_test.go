package cryptor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// helper: generate RSA keys, save to temp files, return paths
func generateTempKeys(t *testing.T) (privPath, pubPath string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	privFile, err := os.CreateTemp("", "private*.pem")
	require.NoError(t, err)
	_, err = privFile.Write(privPem)
	require.NoError(t, err)
	require.NoError(t, privFile.Close())

	pubFile, err := os.CreateTemp("", "public*.pem")
	require.NoError(t, err)
	_, err = pubFile.Write(pubPem)
	require.NoError(t, err)
	require.NoError(t, pubFile.Close())

	return privFile.Name(), pubFile.Name()
}

func TestCryptor_EncryptDecrypt(t *testing.T) {
	privPath, pubPath := generateTempKeys(t)
	defer os.Remove(privPath)
	defer os.Remove(pubPath)

	c, err := New(
		WithPrivateKeyPath(privPath),
		WithPublicKeyPath(pubPath),
	)
	require.NoError(t, err)

	plaintext := []byte("Hello, asymmetric encryption!")

	encrypted, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestCryptor_EncryptWithoutPublicKey(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	_, err = c.Encrypt([]byte("test"))
	require.Error(t, err)
}

func TestCryptor_DecryptWithoutPrivateKey(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	_, err = c.Decrypt([]byte("test"))
	require.Error(t, err)
}

func TestWithPrivateKeyPath_InvalidPath(t *testing.T) {
	_, err := New(WithPrivateKeyPath("nonexistent.pem"))
	require.Error(t, err)
}

func TestWithPublicKeyPath_InvalidPath(t *testing.T) {
	_, err := New(WithPublicKeyPath("nonexistent.pem"))
	require.Error(t, err)
}
