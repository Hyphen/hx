package httputil

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHyphenHTTPClientWithTimeout(t *testing.T) {
	client := NewHyphenHTTPClientWithTimeout(2 * time.Minute)
	assert.Equal(t, 2*time.Minute, client.client.Timeout)
}

func TestNewHyphenHTTPClientUsesDefaultTimeout(t *testing.T) {
	client := NewHyphenHTTPClient()
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestHyphenClientDoReturnsErrorForNilRequest(t *testing.T) {
	client := &HyphenClient{}

	_, err := client.Do(nil)

	assert.EqualError(t, err, "Request is required")
}

func TestHyphenClientDoReturnsErrorForNilRequestURL(t *testing.T) {
	client := &HyphenClient{}

	_, err := client.Do(&http.Request{Method: http.MethodGet})

	assert.EqualError(t, err, "Request URL is required")
}

func TestHyphenClientDoWrapsTransportErrorsWithoutDuplicatingTheUnderlyingError(t *testing.T) {
	setupLocalConfig(t)

	client := &HyphenClient{
		client: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/test", nil)
	require.NoError(t, err)

	_, err = client.Do(req)

	require.Error(t, err)
	assert.EqualError(t, err, "Request failed: POST https://example.com/test")
	assert.ErrorIs(t, err, assert.AnError)
}

func setupLocalConfig(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)

	previousDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	apiKey := "test-api-key"
	require.NoError(t, config.InitializeConfig(config.Config{
		HyphenAPIKey: &apiKey,
	}, config.ManifestConfigFile))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHyphenClientDoReportsConfigErrorsBeforeSendingRequests(t *testing.T) {
	for _, test := range []struct {
		name, local, expected string
		readErr               error
	}{
		{"invalid local JSON", `{"static":{"headers":[]},}`, `Failed to load .hx: Invalid JSON in ".hx" at byte`, nil},
		{"unreadable local config", "", "Failed to load .hx: open .hx: permission denied", &os.PathError{Op: "open", Path: ".hx", Err: os.ErrPermission}},
		{"no local config", "", "", os.ErrNotExist},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := config.FS
			config.FS = &fsutil.MockFileSystem{ReadFileFunc: func(filename string) ([]byte, error) {
				if filename == config.ManifestConfigFile {
					return []byte(test.local), test.readErr
				}
				return []byte(`{"hyphen_api_key":"secret-api-key"}`), nil
			}}
			t.Cleanup(func() { config.FS = original })

			sent := false
			client := &HyphenClient{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				sent = true
				assert.Equal(t, "secret-api-key", req.Header.Get("x-api-key"))
				return &http.Response{StatusCode: http.StatusOK}, nil
			})}}
			req, err := http.NewRequest(http.MethodGet, "https://example.com/api/execution-context/", nil)
			require.NoError(t, err)

			_, err = client.Do(req)

			if test.expected == "" {
				require.NoError(t, err)
				assert.True(t, sent)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expected)
				assert.NotContains(t, err.Error(), "secret-api-key")
				assert.False(t, sent)
				if test.readErr != nil {
					assert.ErrorIs(t, err, test.readErr)
				}
			}
		})
	}
}
