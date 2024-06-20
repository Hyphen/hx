package secretkey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type SecretKey struct {
	secretBase64 string
}

type SecretKeyer interface {
	Base64() string
	HashSHA() string
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
		panic(err)
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
		panic(err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}
