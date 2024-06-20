package secretkey

import (
	"crypto/rand"
	"fmt"
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
	if len(sk.Base64()) != 256 {
		t.Errorf("New() produced Base64 string of length %d, want %d", len(sk.Base64()), 256)
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
	expectedHash := "b400df4d7db31b2ca6d9b69f261017026177d0dafcf1af8b73e3f0bc33210b50" // Expected hash of "secret-key"
	sk := FromBase64(base64Str)
	if sk.HashSHA() != expectedHash {
		t.Errorf("HashSHA() = %v, want %v", sk.HashSHA(), expectedHash)
	}
}

// Test for error handling in New function
func TestNew_ErrorHandling(t *testing.T) {
	// Replace rand.Reader to simulate an error
	oldRandReader := rand.Reader
	defer func() { rand.Reader = oldRandReader }()
	rand.Reader = &errorReader{}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("New() did not panic on rand.Read error")
		}
	}()

	New()
}

// Simulate an error for rand.Reader
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}
