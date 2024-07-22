package secretkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

type SecretKey struct {
	secretBase64 string
}

type SecretKeyer interface {
	Base64() string
	HashSHA() string
	Encrypt(message string) (string, error)
	Decrypt(encryptedMessage string) (string, error)
}

func FromBase64(secretBase64 string) *SecretKey {
	return &SecretKey{
		secretBase64: secretBase64,
	}
}

func New() *SecretKey {

	secret := make([]byte, 256)
	if _, err := rand.Read(secret); err != nil {
		fmt.Println("Error generating secret key:", err)
		os.Exit(1)
	}

	secretBase64 := base64.StdEncoding.EncodeToString(secret)
	if len(secretBase64) > 256 {
		secretBase64 = secretBase64[:256]
	}

	return &SecretKey{
		secretBase64: secretBase64,
	}
}

func (s SecretKey) Base64() string {
	return s.secretBase64
}

func (s SecretKey) HashSHA() string {
	hasher := sha256.New()
	if _, err := hasher.Write([]byte(s.secretBase64)); err != nil {
		fmt.Println("Error hashing secret key:", err)
		os.Exit(1)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (s SecretKey) Encrypt(message string) (string, error) {
	key := []byte(s.HashSHA())[:32] // AES requires a key of 16, 24, or 32 bytes

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(message)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func (s SecretKey) Decrypt(encryptedMessage string) (string, error) {
	key := []byte(s.HashSHA())[:32] // AES requires a key of 16, 24, or 32 bytes

	ciphertext, err := base64.URLEncoding.DecodeString(encryptedMessage)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}
