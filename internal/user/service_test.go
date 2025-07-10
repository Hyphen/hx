package user

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetExecutionContext(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*httputil.MockHTTPClient)
		expectedError  string
		expectedUserID string
	}{
		{
			name: "Successful request",
			setupMocks: func(mhc *httputil.MockHTTPClient) {
				executionContext := models.ExecutionContext{
					Member: models.Member{
						ID: "test_member_id",
					},
					User: models.User{
						ID: "test_user_id",
					},
				}
				body, _ := json.Marshal(executionContext)
				mhc.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil)
			},
			expectedUserID: "test_user_id",
		},
		{
			name: "HTTP request error",
			setupMocks: func(mhc *httputil.MockHTTPClient) {
				mhc.On("Do", mock.Anything).Return((*http.Response)(nil), errors.New("HTTP error"))
			},
			expectedError: "HTTP error",
		},
		{
			name: "Non-200 status code",
			setupMocks: func(mhc *httputil.MockHTTPClient) {
				mhc.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte("Bad Request"))),
				}, nil)
			},
			expectedError: "bad request: Bad Request",
		},
		{
			name: "Invalid JSON response",
			setupMocks: func(mhc *httputil.MockHTTPClient) {
				mhc.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
				}, nil)
			},
			expectedError: "Failed to process the server response. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := new(httputil.MockHTTPClient)
			tt.setupMocks(mockHTTP)

			us := &UserService{
				baseUrl: "https://test-api.hyphen.ai",
				client:  mockHTTP,
			}

			userInfo, err := us.GetExecutionContext()

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, userInfo.User.ID)
			}

			mockHTTP.AssertExpectations(t)
		})
	}
}

func TestNewService(t *testing.T) {
	originalDev := os.Getenv("HYPHEN_DEV")
	defer os.Setenv("HYPHEN_DEV", originalDev)

	tests := []struct {
		name            string
		customDevValue  string
		expectedBaseURL string
	}{
		{
			name:            "Default base URL",
			customDevValue:  "",
			expectedBaseURL: "https://api.hyphen.ai",
		},
		{
			name:            "Custom base URL",
			customDevValue:  "true",
			expectedBaseURL: "https://dev-api.hyphen.ai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HYPHEN_DEV", tt.customDevValue)
			service := NewService()
			userService, ok := service.(*UserService)
			assert.True(t, ok, "Expected *user.UserService")
			assert.Equal(t, tt.expectedBaseURL, userService.baseUrl)
		})
	}
}
