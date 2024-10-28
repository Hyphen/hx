package zelda

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of the httputil.Client interface
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewService(t *testing.T) {
	service := NewService()
	assert.NotNil(t, service)
	assert.Contains(t, service.baseUrl, "/api")
	assert.NotNil(t, service.httpClient)
}

func TestCreateCode(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	testCode := Code{LongURL: "https://example.com", Domain: "short.ly"}
	responseBody := `{"long_url": "https://example.com", "domain": "short.ly", "code": "abc123"}`

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	createdCode, err := service.CreateCode("org123", testCode)

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", createdCode.LongURL)
	assert.Equal(t, "short.ly", createdCode.Domain)
	assert.Equal(t, "abc123", *createdCode.Code)

	mockHTTPClient.AssertExpectations(t)
}

func TestListDomains(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	responseBody := `{
		"data": [
			{"id": "domain1", "domain": "short.ly", "status": "active"},
			{"id": "domain2", "domain": "tiny.url", "status": "pending"}
		]
	}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	domains, err := service.ListDomains("org123", 10, 1)

	assert.NoError(t, err)
	assert.Len(t, domains, 2)
	assert.Equal(t, "domain1", domains[0].ID)
	assert.Equal(t, "short.ly", domains[0].Domain)
	assert.Equal(t, "active", domains[0].Status)

	mockHTTPClient.AssertExpectations(t)
}

func TestCreateQRCode(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	responseBody := `{
		"id": "qr123",
		"qrCode": "base64encodedimage",
		"qrLink": "https://qr.example.com/qr123"
	}`

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	qr, err := service.CreateQRCode("org123", "code456")

	assert.NoError(t, err)
	assert.Equal(t, "qr123", qr.ID)
	assert.Equal(t, "base64encodedimage", qr.QRCode)
	assert.Equal(t, "https://qr.example.com/qr123", qr.QRLink)

	mockHTTPClient.AssertExpectations(t)
}

func TestErrorHandling(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{"BadRequest", http.StatusBadRequest, "Invalid input", "bad request: Invalid input"},
		{"Unauthorized", http.StatusUnauthorized, "", "unauthorized: please authenticate with `auth` and try again"},
		{"Forbidden", http.StatusForbidden, "", "forbidden: you don't have permission to perform this action"},
		{"NotFound", http.StatusNotFound, "", "not found: "},
		{"Conflict", http.StatusConflict, "Resource already exists", "conflict: Resource already exists"},
		{"TooManyRequests", http.StatusTooManyRequests, "", "rate limit exceeded: please try again later"},
		{"InternalServerError", http.StatusInternalServerError, "", "internal server error: please try again later"},
		{"UnexpectedError", 499, "Custom error", "unexpected error (status code 499): Custom error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockResponse := &http.Response{
				StatusCode: tc.statusCode,
				Body:       io.NopCloser(strings.NewReader(tc.responseBody)),
			}

			mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil).Once()

			_, err := service.CreateCode("org123", Code{})
			assert.Error(t, err)
			assert.Equal(t, tc.expectedErrMsg, err.Error())
		})
	}
}

func TestJSONParsingError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader("Invalid JSON")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	_, err := service.CreateCode("org123", Code{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse JSON response")
}

func TestRequestCreationError(t *testing.T) {
	service := &ZeldaService{
		baseUrl: "://invalid-url",
	}

	_, err := service.CreateCode("org123", Code{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}

func TestReadBodyError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	service := &ZeldaService{
		baseUrl:    "https://api.example.com/link",
		httpClient: mockHTTPClient,
	}

	errorReader := &ErrorReader{Err: errors.New("read error")}
	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(errorReader),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	_, err := service.CreateCode("org123", Code{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")
}

// ErrorReader is a custom io.Reader that always returns an error
type ErrorReader struct {
	Err error
}

func (er *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, er.Err
}

