package members

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewService(t *testing.T) {
	os.Setenv("HYPHEN_CUSTOM_APIX", "https://custom-api.example.com")
	defer os.Unsetenv("HYPHEN_CUSTOM_APIX")

	service := NewService()
	assert.Equal(t, "https://custom-api.example.com", service.baseUrl)
	assert.NotNil(t, service.oauthService)
	assert.NotNil(t, service.httpClient)
}

func TestListMembers(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	responseBody := `{
		"data": [
			{"id": "member1", "firstName": "John", "lastName": "Doe", "email": "john@example.com"},
			{"id": "member2", "firstName": "Jane", "lastName": "Doe", "email": "jane@example.com"}
		]
	}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	members, err := service.ListMembers("org1")

	assert.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, "member1", members[0].ID)
	assert.Equal(t, "member2", members[1].ID)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestCreateMemberForOrg(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	newMember := Member{
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     "alice@example.com",
	}

	responseBody := `{"id": "new_member", "firstName": "Alice", "lastName": "Smith", "email": "alice@example.com"}`

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	createdMember, err := service.CreateMemberForOrg("org1", newMember)

	assert.NoError(t, err)
	assert.Equal(t, "new_member", createdMember.ID)
	assert.Equal(t, "Alice", createdMember.FirstName)
	assert.Equal(t, "Smith", createdMember.LastName)
	assert.Equal(t, "alice@example.com", createdMember.Email)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestErrorHandling(t *testing.T) {
	t.Run("Authentication Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &MemberService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("", errors.New("auth error"))

		_, err := service.ListMembers("org1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to authenticate")
	})

	t.Run("HTTP Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &MemberService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("valid_token", nil)

		mockResponse := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.ListMembers("org1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error: please try again later")
	})

	t.Run("JSON Parsing Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &MemberService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("valid_token", nil)

		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("Invalid JSON")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.ListMembers("org1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to parse JSON response")
	})
}

func TestMemberService_HTTPClientError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)
	mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("error")),
	}, errors.New("network error"))

	_, err := service.ListMembers("org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")

	_, err = service.CreateMemberForOrg("org1", Member{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")
}

func TestMemberService_ReadBodyError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	errorReader := &ErrorReader{Err: errors.New("read error")}
	mockResponseGet := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(errorReader),
	}

	mockResponseCreate := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(errorReader),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponseGet, nil).Once()
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponseCreate, nil).Once()

	_, err := service.ListMembers("org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")

	_, err = service.CreateMemberForOrg("org1", Member{})
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

func TestMemberService_NewRequestError(t *testing.T) {
	service := &MemberService{
		baseUrl: "://invalid-url",
	}

	mockOAuthService := new(oauth.MockOAuthService)
	service.oauthService = mockOAuthService
	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	_, err := service.ListMembers("org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.CreateMemberForOrg("org1", Member{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}

func TestDeleteMember(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	mockResponse := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteMember("org1", "member1")

	assert.NoError(t, err)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestDeleteMember_Error(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	mockResponse := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteMember("org1", "member1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error: please try again later")

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestMemberService_DeleteMember_HTTPClientError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &MemberService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	// Return a non-nil http.Response with the error
	mockResponse := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("network error")),
	}
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, errors.New("network error"))

	err := service.DeleteMember("org1", "member1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")
}

func TestMemberService_DeleteMember_NewRequestError(t *testing.T) {
	service := &MemberService{
		baseUrl: "://invalid-url",
	}

	mockOAuthService := new(oauth.MockOAuthService)
	service.oauthService = mockOAuthService
	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	err := service.DeleteMember("org1", "member1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}
