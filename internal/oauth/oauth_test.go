package oauth

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Hyphen/cli/internal/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockTimeProvider is a mock implementation of the TimeProvider interface
type MockTimeProvider struct {
	mock.Mock
}

func (m *MockTimeProvider) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func TestDefaultOAuthService(t *testing.T) {
	service := DefaultOAuthService()
	assert.NotNil(t, service)
	assert.IsType(t, &http.Client{}, service.httpClient)
	assert.IsType(t, &timeprovider.RealTimeProvider{}, service.timeProvider)
	assert.NotNil(t, service.browserOpener)
}

func TestNewOAuthService(t *testing.T) {
	httpClient := &MockHTTPClient{}
	timeProvider := timeprovider.NewMockTimeProvider()
	browserOpener := func(url string) error { return nil }
	service := NewOAuthService(httpClient, timeProvider, browserOpener, rand.Reader)
	assert.NotNil(t, service)
	assert.Equal(t, httpClient, service.httpClient)
	assert.Equal(t, timeProvider, service.timeProvider)
	assert.NotNil(t, service.browserOpener)
}

func TestGeneratePKCE(t *testing.T) {
	service := DefaultOAuthService()
	verifier, challenge, err := service.generatePKCE()
	assert.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.NotEmpty(t, challenge)
	assert.NotEqual(t, verifier, challenge)
}

func TestExchangeCodeForToken(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(mockClient, mockTime, func(url string) error { return nil }, rand.Reader)

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBufferString(`{
			"access_token": "test_access_token",
			"refresh_token": "test_refresh_token",
			"id_token": "test_id_token",
			"expires_in": 3600
		}`)),
	}
	mockClient.On("Do", mock.Anything).Return(mockResp, nil)
	mockTime.On("Now").Return(time.Unix(1000000000, 0))

	token, err := service.exchangeCodeForToken("test_code", "test_verifier")
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test_access_token", token.AccessToken)
	assert.Equal(t, "test_refresh_token", token.RefreshToken)
	assert.Equal(t, "test_id_token", token.IDToken)
	assert.Equal(t, 3600, token.ExpiresIn)
	assert.Equal(t, int64(1000003600), token.ExpiryTime)

	mockClient.AssertExpectations(t)
	mockTime.AssertExpectations(t)
}

func TestIsTokenExpired(t *testing.T) {
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(&http.Client{}, mockTime, func(url string) error { return nil }, rand.Reader)

	mockTime.On("Now").Return(time.Unix(1000000000, 0))

	assert.True(t, service.IsTokenExpired(999999999))
	assert.False(t, service.IsTokenExpired(1000000001))

	mockTime.AssertExpectations(t)
}

func TestRefreshToken(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(mockClient, mockTime, func(url string) error { return nil }, rand.Reader)

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBufferString(`{
			"access_token": "new_access_token",
			"refresh_token": "new_refresh_token",
			"id_token": "new_id_token",
			"expires_in": 3600
		}`)),
	}
	mockClient.On("Do", mock.Anything).Return(mockResp, nil)
	mockTime.On("Now").Return(time.Unix(1000000000, 0))

	token, err := service.RefreshToken("old_refresh_token")
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "new_access_token", token.AccessToken)
	assert.Equal(t, "new_refresh_token", token.RefreshToken)
	assert.Equal(t, "new_id_token", token.IDToken)
	assert.Equal(t, 3600, token.ExpiresIn)
	assert.Equal(t, int64(1000003600), token.ExpiryTime)

	mockClient.AssertExpectations(t)
	mockTime.AssertExpectations(t)
}

func TestStartOAuthServer(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()

	// Create a channel to signal when the browser opener has been called
	browserOpenerCalled := make(chan bool, 1)

	// Create a mock browser opener that signals when it's called
	mockBrowserOpener := func(url string) error {
		browserOpenerCalled <- true
		return nil
	}

	service := NewOAuthService(mockClient, mockTime, mockBrowserOpener, rand.Reader)

	// Mock the exchange code for token response
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBufferString(`{
			"access_token": "test_access_token",
			"refresh_token": "test_refresh_token",
			"id_token": "test_id_token",
			"expires_in": 3600
		}`)),
	}
	mockClient.On("Do", mock.Anything).Return(mockResp, nil)
	mockTime.On("Now").Return(time.Unix(1000000000, 0))

	// Run StartOAuthServer in a goroutine
	tokenChan := make(chan *TokenResponse)
	errChan := make(chan error)
	go func() {
		token, err := service.StartOAuthServer()
		if err != nil {
			errChan <- err
		} else {
			tokenChan <- token
		}
	}()

	// Wait for the browser opener to be called
	select {
	case <-browserOpenerCalled:
		// Simulate the OAuth callback
		go func() {
			resp, err := http.Get("http://localhost:5001/token?code=test_code")
			if err != nil {
				t.Logf("Error simulating OAuth callback: %v", err)
			}
			defer resp.Body.Close()
		}()
	case <-time.After(2 * time.Second):
		t.Fatal("Browser opener was not called")
	}

	// Wait for the result
	select {
	case token := <-tokenChan:
		assert.NotNil(t, token)
		assert.Equal(t, "test_access_token", token.AccessToken)
		assert.Equal(t, "test_refresh_token", token.RefreshToken)
		assert.Equal(t, "test_id_token", token.IDToken)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.Equal(t, int64(1000003600), token.ExpiryTime)
	case err := <-errChan:
		t.Fatalf("StartOAuthServer failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}

	mockClient.AssertExpectations(t)
	mockTime.AssertExpectations(t)
}

type MockRandReader struct {
	mock.Mock
}

func (m *MockRandReader) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func TestGeneratePKCE_Error(t *testing.T) {
	mockReader := new(MockRandReader)
	mockReader.On("Read", mock.Anything).Return(0, fmt.Errorf("the fake error"))

	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(mockClient, mockTime, func(url string) error { return nil }, mockReader)

	_, _, err := service.generatePKCE()

	assert.EqualError(t, err, "Failed to generate PKCE code verifier: the fake error")
	mockReader.AssertExpectations(t)
}

func TestExchangeCodeForToken_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(mockClient, mockTime, func(url string) error { return nil }, rand.Reader)

	mockResp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(`{"error": "invalid_request"}`)),
	}
	mockClient.On("Do", mock.Anything).Return(mockResp, nil)

	token, err := service.exchangeCodeForToken("test_code", "test_verifier")
	assert.Error(t, err)
	assert.Nil(t, token)

	mockClient.AssertExpectations(t)
}

func TestRefreshToken_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockTime := timeprovider.NewMockTimeProvider()
	service := NewOAuthService(mockClient, mockTime, func(url string) error { return nil }, rand.Reader)

	mockResp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(`{"error": "invalid_grant"}`)),
	}
	mockClient.On("Do", mock.Anything).Return(mockResp, nil)

	token, err := service.RefreshToken("old_refresh_token")
	assert.Error(t, err)
	assert.Nil(t, token)

	mockClient.AssertExpectations(t)
}
