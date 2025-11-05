package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// Crypto handles encryption and decryption operations using AES-GCM
type Crypto struct {
	gcm cipher.AEAD
}

func NewCrypto(secret string) (*Crypto, error) {
	if secret == "" {
		return nil, fmt.Errorf("secret cannot be empty")
	}

	// Generate a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	return &Crypto{gcm: gcm}, nil
}

// Encrypt encrypts the plaintext data using AES-GCM
func (c *Crypto) Encrypt(plaintext []byte) (string, error) {
	// Generate random nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt and authenticate data
	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the encrypted string using AES-GCM
func (c *Crypto) Decrypt(encryptedStr string) ([]byte, error) {
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// Check if the ciphertext is too short
	if len(ciphertext) < c.gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data length")
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:c.gcm.NonceSize()]
	ciphertext = ciphertext[c.gcm.NonceSize():]

	// Decrypt and verify data
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed")
	}

	return plaintext, nil
}
