package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	_ "github.com/joho/godotenv/autoload"
	"io"
	"os"
)

// cryptoKey is read per call, not at package load, so tests can set
// CRYPTO_KEY with t.Setenv.
func cryptoKey() []byte { return []byte(os.Getenv("CRYPTO_KEY")) }

func Encrypt(plaintext string) (string, error) { return EncryptWithKey(cryptoKey(), plaintext) }

func Decrypt(ciphertext string) (string, error) { return DecryptWithKey(cryptoKey(), ciphertext) }

// EncryptWithKey and DecryptWithKey take the key explicitly, for key rotation
// (cmd/rotatekey), where two keys are in play at once.
func EncryptWithKey(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptWithKey(key []byte, ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce := data[:gcm.NonceSize()]
	encrypted := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
