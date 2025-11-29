package gittokens

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

var (
	ErrInvalidKey    = errors.New("encryption key must be 32 bytes")
	ErrInvalidCipher = errors.New("invalid ciphertext")
	ErrMissingKey    = errors.New("CONFIG_ENCRYPTION_KEY environment variable not set")
)

// getEncryptionKey returns the encryption key from environment
func getEncryptionKey() ([]byte, error) {
	key := os.Getenv("CONFIG_ENCRYPTION_KEY")
	if key == "" {
		return nil, ErrMissingKey
	}

	// Decode from base64 if it looks like base64
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	// Otherwise use raw bytes (must be exactly 32 bytes for AES-256)
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return []byte(key), nil
}

// encrypt encrypts plaintext using AES-256-GCM
func encrypt(plaintext string) (string, error) {
	key, err := getEncryptionKey()
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

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-256-GCM
func decrypt(ciphertext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

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

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCipher
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
