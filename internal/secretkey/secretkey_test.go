package secretkey

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromBase64(t *testing.T) {
	base64String := "dGVzdFNlY3JldEtleQ=="
	sk := FromBase64(base64String)
	assert.Equal(t, base64String, sk.Base64())
}

func TestNew(t *testing.T) {
	sk, err := New()
	assert.NoError(t, err)
	assert.NotEmpty(t, sk.Base64())
	assert.LessOrEqual(t, len(sk.Base64()), 256)

	// Test if the generated key is valid base64
	_, err = base64.StdEncoding.DecodeString(sk.Base64())
	assert.NoError(t, err)
}

func TestSecretKey_Base64(t *testing.T) {
	base64String := "dGVzdFNlY3JldEtleQ=="
	sk := FromBase64(base64String)
	assert.Equal(t, base64String, sk.Base64())
}

func TestSecretKey_HashSHA(t *testing.T) {
	sk := FromBase64("dGVzdFNlY3JldEtleQ==")
	hash, err := sk.HashSHA()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hash is 64 characters long in hex format
}

func TestSecretKey_Encrypt(t *testing.T) {
	sk := FromBase64("dGVzdFNlY3JldEtleQ==")
	message := "Hello, World!"
	encrypted, err := sk.Encrypt(message)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, message, encrypted)

	// Test if the encrypted message is valid base64
	_, err = base64.URLEncoding.DecodeString(encrypted)
	assert.NoError(t, err)
}

func TestSecretKey_Decrypt(t *testing.T) {
	sk := FromBase64("dGVzdFNlY3JldEtleQ==")
	message := "Hello, World!"
	encrypted, err := sk.Encrypt(message)
	assert.NoError(t, err)

	decrypted, err := sk.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, message, decrypted)
}

func TestSecretKey_EncryptDecrypt_LongMessage(t *testing.T) {
	sk := FromBase64("dGVzdFNlY3JldEtleQ==")
	message := strings.Repeat("Long message. ", 100)
	encrypted, err := sk.Encrypt(message)
	assert.NoError(t, err)

	decrypted, err := sk.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, message, decrypted)
}

func TestSecretKey_Decrypt_InvalidInput(t *testing.T) {
	sk := FromBase64("dGVzdFNlY3JldEtleQ==")

	_, err := sk.Decrypt("invalid base64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to decode encrypted message")

	_, err = sk.Decrypt("dGVzdA==") // "test" in base64, which is too short
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Ciphertext is too short")
}
