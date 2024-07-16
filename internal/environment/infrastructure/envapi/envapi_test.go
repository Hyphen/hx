package envapi

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of the http.Client
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockOAuthService is a mock implementation of the oauth.OAuthServiceInterface
type MockOAuthService struct {
	mock.Mock
}

func (m *MockOAuthService) IsTokenExpired(expiryTime int64) bool {
	args := m.Called(expiryTime)
	return args.Bool(0)
}

func (m *MockOAuthService) RefreshToken(refreshToken string) (*oauth.TokenResponse, error) {
	args := m.Called(refreshToken)
	return args.Get(0).(*oauth.TokenResponse), args.Error(1)
}

func TestNew(t *testing.T) {
	api := New()
	assert.NotNil(t, api)
	assert.Equal(t, "http://localhost:4001", api.baseUrl)
	assert.NotNil(t, api.httpClient)
	assert.NotNil(t, api.oauthService)
	assert.NotNil(t, api.configLoader)
	assert.NotNil(t, api.configSaver)
}

func TestGetAuthToken(t *testing.T) {
	mockOAuth := new(MockOAuthService)
	api := &EnvApi{
		oauthService: mockOAuth,
		configLoader: func() (config.CredentialsConfig, error) {
			return config.CredentialsConfig{
				Default: config.Credentials{
					HyphenAccessToken:  "test_access_token",
					HyphenRefreshToken: "test_refresh_token",
					ExpiryTime:         1000000000,
				},
			}, nil
		},
		configSaver: func(string, string, string, int64) error { return nil },
	}

	mockOAuth.On("IsTokenExpired", int64(1000000000)).Return(false)

	token, err := api.getAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "test_access_token", token)

	mockOAuth.AssertExpectations(t)
}

func TestGetAuthToken_Refresh(t *testing.T) {
	mockOAuth := new(MockOAuthService)
	api := &EnvApi{
		oauthService: mockOAuth,
		configLoader: func() (config.CredentialsConfig, error) {
			return config.CredentialsConfig{
				Default: config.Credentials{
					HyphenAccessToken:  "old_access_token",
					HyphenRefreshToken: "test_refresh_token",
					ExpiryTime:         1000000000,
				},
			}, nil
		},
		configSaver: func(string, string, string, int64) error { return nil },
	}

	mockOAuth.On("IsTokenExpired", int64(1000000000)).Return(true)
	mockOAuth.On("RefreshToken", "test_refresh_token").Return(&oauth.TokenResponse{
		AccessToken:  "new_access_token",
		RefreshToken: "new_refresh_token",
		IDToken:      "new_id_token",
		ExpiryTime:   2000000000,
	}, nil)

	token, err := api.getAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "new_access_token", token)

	mockOAuth.AssertExpectations(t)
}

func TestInitialize(t *testing.T) {
	mockHTTP := new(MockHTTPClient)
	mockOAuth := new(MockOAuthService)
	api := &EnvApi{
		baseUrl:      "http://test.com",
		httpClient:   mockHTTP,
		oauthService: mockOAuth,
		configLoader: func() (config.CredentialsConfig, error) {
			return config.CredentialsConfig{
				Default: config.Credentials{
					HyphenAccessToken: "test_access_token",
					ExpiryTime:        1000000000,
				},
			}, nil
		},
	}

	mockOAuth.On("IsTokenExpired", int64(1000000000)).Return(false)

	mockResp := &http.Response{
		StatusCode: 201,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
	}
	mockHTTP.On("Do", mock.Anything).Return(mockResp, nil)

	err := api.Initialize("test_api", "test_id")
	assert.NoError(t, err)

	mockHTTP.AssertExpectations(t)
	mockOAuth.AssertExpectations(t)
}

func TestUploadEnvVariable(t *testing.T) {
	mockHTTP := new(MockHTTPClient)
	mockOAuth := new(MockOAuthService)
	api := &EnvApi{
		baseUrl:      "http://test.com",
		httpClient:   mockHTTP,
		oauthService: mockOAuth,
		configLoader: func() (config.CredentialsConfig, error) {
			return config.CredentialsConfig{
				Default: config.Credentials{
					HyphenAccessToken: "test_access_token",
					ExpiryTime:        1000000000,
				},
			}, nil
		},
	}

	mockOAuth.On("IsTokenExpired", int64(1000000000)).Return(false)

	mockResp := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
	}
	mockHTTP.On("Do", mock.Anything).Return(mockResp, nil)

	envData := envvars.EnviromentVarsData{
		Size:           "10",
		CountVariables: 5,
		Data:           "encrypted_data",
	}

	err := api.UploadEnvVariable("test_env", "test_app_id", envData)
	assert.NoError(t, err)

	mockHTTP.AssertExpectations(t)
	mockOAuth.AssertExpectations(t)
}

func TestGetEncryptedVariables(t *testing.T) {
	mockHTTP := new(MockHTTPClient)
	mockOAuth := new(MockOAuthService)
	api := &EnvApi{
		baseUrl:      "http://test.com",
		httpClient:   mockHTTP,
		oauthService: mockOAuth,
		configLoader: func() (config.CredentialsConfig, error) {
			return config.CredentialsConfig{
				Default: config.Credentials{
					HyphenAccessToken: "test_access_token",
					ExpiryTime:        1000000000,
				},
			}, nil
		},
	}

	mockOAuth.On("IsTokenExpired", int64(1000000000)).Return(false)

	envData := envvars.EnviromentVarsData{
		Size:           "10",
		CountVariables: 5,
		Data:           "encrypted_data",
	}
	respBody, _ := json.Marshal(envData)

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBuffer(respBody)),
	}
	mockHTTP.On("Do", mock.Anything).Return(mockResp, nil)

	data, err := api.GetEncryptedVariables("test_env", "test_app_id")
	assert.NoError(t, err)
	assert.Equal(t, "encrypted_data", data)

	mockHTTP.AssertExpectations(t)
	mockOAuth.AssertExpectations(t)
}
