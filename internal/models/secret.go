package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/Hyphen/cli/pkg/errors"
)

type Secret struct {
	SecretKeyId     int64  `json:"secret_key_id"`
	Base64SecretKey string `json:"secret_key"`
}

// Base64 returns the base64 encoded secret key
func (s Secret) Base64() string {
	return s.Base64SecretKey
}

// HashSHA returns the SHA256 hash of the secret key
func (s Secret) HashSHA() (string, error) {
	hasher := sha256.New()
	if _, err := hasher.Write([]byte(s.Base64SecretKey)); err != nil {
		return "", errors.Wrap(err, "Failed to hash secret key")
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Encrypt encrypts a message using AES encryption
func (s Secret) Encrypt(message string) (string, error) {
	hashSHA, err := s.HashSHA()
	if err != nil {
		return "", err
	}
	key := []byte(hashSHA)[:32] // AES requires a key of 16, 24, or 32 bytes

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create cipher block")
	}

	plaintext := []byte(message)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", errors.Wrap(err, "Failed to generate initialization vector")
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts an encrypted message using AES decryption
func (s Secret) Decrypt(encryptedMessage string) (string, error) {
	hashSHA, err := s.HashSHA()
	if err != nil {
		return "", err
	}
	key := []byte(hashSHA)[:32] // AES requires a key of 16, 24, or 32 bytes

	ciphertext, err := base64.URLEncoding.DecodeString(encryptedMessage)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decode encrypted message")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create cipher block")
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("Ciphertext is too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// NewSecret creates a new Secret from a base64 encoded secret key
func NewSecret(secretBase64 string) Secret {
	return Secret{
		SecretKeyId:     time.Now().Unix(),
		Base64SecretKey: secretBase64,
	}
}

// GenerateSecret creates a new Secret with a randomly generated key
func GenerateSecret() (Secret, error) {
	secret := make([]byte, 256)
	if _, err := rand.Read(secret); err != nil {
		return Secret{}, errors.Wrap(err, "Failed to generate secret key")
	}

	secretBase64 := base64.StdEncoding.EncodeToString(secret)
	if len(secretBase64) > 256 {
		secretBase64 = secretBase64[:256]
	}

	return Secret{
		SecretKeyId:     time.Now().Unix(),
		Base64SecretKey: secretBase64,
	}, nil
}
