package secretkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/Hyphen/cli/pkg/errors"
)

type SecretKey struct {
	secretBase64 string
}

type SecretKeyer interface {
	Base64() string
	HashSHA() (string, error)
	Encrypt(message string) (string, error)
	Decrypt(encryptedMessage string) (string, error)
}

func FromBase64(secretBase64 string) *SecretKey {
	return &SecretKey{
		secretBase64: secretBase64,
	}
}

func New() (*SecretKey, error) {
	secret := make([]byte, 256)
	if _, err := rand.Read(secret); err != nil {
		return nil, errors.Wrap(err, "Failed to generate secret key")
	}

	secretBase64 := base64.StdEncoding.EncodeToString(secret)
	if len(secretBase64) > 256 {
		secretBase64 = secretBase64[:256]
	}

	return &SecretKey{
		secretBase64: secretBase64,
	}, nil
}

func (s SecretKey) Base64() string {
	return s.secretBase64
}

func (s SecretKey) HashSHA() (string, error) {
	hasher := sha256.New()
	if _, err := hasher.Write([]byte(s.secretBase64)); err != nil {
		return "", errors.Wrap(err, "Failed to hash secret key")
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (s SecretKey) Encrypt(message string) (string, error) {
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

func (s SecretKey) Decrypt(encryptedMessage string) (string, error) {
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
