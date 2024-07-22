package secretkey

import (
	"encoding/base64"
	"testing"
)

func TestFromBase64(t *testing.T) {
	base64Str := "c2VjcmV0LWtleQ=="
	sk := FromBase64(base64Str)
	if sk.Base64() != base64Str {
		t.Errorf("FromBase64() = %v, want %v", sk.Base64(), base64Str)
	}
}

func TestNew(t *testing.T) {
	sk := New()
	if len(sk.Base64()) == 0 {
		t.Errorf("New() produced empty Base64 string")
	}
}

func TestBase64(t *testing.T) {
	base64Str := "c2VjcmV0LWtleQ=="
	sk := FromBase64(base64Str)
	if sk.Base64() != base64Str {
		t.Errorf("Base64() = %v, want %v", sk.Base64(), base64Str)
	}
}

func TestHashSHA(t *testing.T) {
	base64Str := "c2VjcmV0LWtleQ=="
	expectedHash := "b400df4d7db31b2ca6d9b69f261017026177d0dafcf1af8b73e3f0bc33210b50" // Expected hash of "secret-key in Base64"
	sk := FromBase64(base64Str)
	if sk.HashSHA() != expectedHash {
		t.Errorf("HashSHA() = %v, want %v", sk.HashSHA(), expectedHash)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	sk := New()

	message := "This is a test message."
	encrypted, err := sk.Encrypt(message)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := sk.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != message {
		t.Errorf("Decrypt() = %v, want %v", decrypted, message)
	}
}

// TestDecryptErrorHandling tests error handling in the Decrypt function
func TestDecryptErrorHandling(t *testing.T) {
	sk := New()

	// Test with an improperly formatted base64 string
	_, err := sk.Decrypt("bad-base64")
	if err == nil {
		t.Error("Decrypt() expected error, got nil")
	}

	// Test with ciphertext that is too short
	_, err = sk.Decrypt(base64.URLEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Error("Decrypt() expected 'ciphertext too short' error, got nil")
	}
}
