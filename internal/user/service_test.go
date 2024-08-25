package user

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetUserInformation(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*oauth.MockOAuthService, *httputil.MockHTTPClient)
		expectedError  string
		expectedUserID string
	}{
		{
			name: "Successful request",
			setupMocks: func(mos *oauth.MockOAuthService, mhc *httputil.MockHTTPClient) {
				mos.On("GetValidToken").Return("valid_token", nil)
				userInfo := UserInfo{
					DecodedIdToken: TokenInfo{
						Sub: "test_user_id",
					},
				}
				body, _ := json.Marshal(userInfo)
				mhc.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil)
			},
			expectedUserID: "test_user_id",
		},
		{
			name: "OAuth token error",
			setupMocks: func(mos *oauth.MockOAuthService, mhc *httputil.MockHTTPClient) {
				mos.On("GetValidToken").Return("", errors.New("OAuth error"))
			},
			expectedError: "Failed to authenticate. Please check your credentials and try again.",
		},
		{
			name: "HTTP request error",
			setupMocks: func(mos *oauth.MockOAuthService, mhc *httputil.MockHTTPClient) {
				mos.On("GetValidToken").Return("valid_token", nil)
				mhc.On("Do", mock.Anything).Return((*http.Response)(nil), errors.New("HTTP error"))
			},
			expectedError: "Failed to connect to the server. Please check your internet connection and try again.",
		},
		{
			name: "Non-200 status code",
			setupMocks: func(mos *oauth.MockOAuthService, mhc *httputil.MockHTTPClient) {
				mos.On("GetValidToken").Return("valid_token", nil)
				mhc.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte("Bad Request"))),
				}, nil)
			},
			expectedError: "bad request: Bad Request",
		},
		{
			name: "Invalid JSON response",
			setupMocks: func(mos *oauth.MockOAuthService, mhc *httputil.MockHTTPClient) {
				mos.On("GetValidToken").Return("valid_token", nil)
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
			mockOAuth := new(oauth.MockOAuthService)
			mockHTTP := new(httputil.MockHTTPClient)
			tt.setupMocks(mockOAuth, mockHTTP)

			us := &UserService{
				baseUrl:      "https://test-api.hyphen.ai",
				oauthService: mockOAuth,
				client:       mockHTTP,
			}

			userInfo, err := us.GetUserInformation()

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, userInfo.DecodedIdToken.Sub)
			}

			mockOAuth.AssertExpectations(t)
			mockHTTP.AssertExpectations(t)
		})
	}
}

func TestNewService(t *testing.T) {
	originalEnv := os.Getenv("HYPHEN_CUSTOM_APIX")
	defer os.Setenv("HYPHEN_CUSTOM_APIX", originalEnv)

	tests := []struct {
		name            string
		customAPIValue  string
		expectedBaseURL string
	}{
		{
			name:            "Default base URL",
			customAPIValue:  "",
			expectedBaseURL: "https://dev-api.hyphen.ai",
		},
		{
			name:            "Custom base URL",
			customAPIValue:  "https://custom-api.hyphen.ai",
			expectedBaseURL: "https://custom-api.hyphen.ai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HYPHEN_CUSTOM_APIX", tt.customAPIValue)
			service := NewService()
			userService, ok := service.(*UserService)
			assert.True(t, ok, "Expected *user.UserService")
			assert.Equal(t, tt.expectedBaseURL, userService.baseUrl)
		})
	}
}
