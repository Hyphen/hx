package httputil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHyphenHTTPClientWithTimeout(t *testing.T) {
	client := NewHyphenHTTPClientWithTimeout(2 * time.Minute)
	assert.Equal(t, 2*time.Minute, client.client.Timeout)
}

func TestNewHyphenHTTPClientUsesDefaultTimeout(t *testing.T) {
	client := NewHyphenHTTPClient()
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}
