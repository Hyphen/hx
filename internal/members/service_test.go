package members

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewService(t *testing.T) {
	os.Setenv("HYPHEN_DEV", "true")
	defer os.Unsetenv("HYPHEN_DEV")

	service := NewService()
	assert.Equal(t, "https://dev-api.hyphen.ai", service.baseUrl)
	assert.NotNil(t, service.httpClient)
}

func TestListMembers(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

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

	mockHTTPClient.AssertExpectations(t)
}

func TestCreateMemberForOrg(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

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

	mockHTTPClient.AssertExpectations(t)
}

func TestDeleteMember(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	mockResponse := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteMember("org1", "member1")

	assert.NoError(t, err)

	mockHTTPClient.AssertExpectations(t)
}

func TestDeleteMember_Error(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	mockResponse := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteMember("org1", "member1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error: please try again later")

	mockHTTPClient.AssertExpectations(t)
}

func TestErrorHandling(t *testing.T) {
	t.Run("HTTP Error", func(t *testing.T) {
		mockHTTPClient := new(MockHTTPClient)

		service := &MemberService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

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
		mockHTTPClient := new(MockHTTPClient)

		service := &MemberService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

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
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("error")),
	}, errors.New("network error"))

	_, err := service.ListMembers("org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")

	_, err = service.CreateMemberForOrg("org1", Member{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestMemberService_ReadBodyError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	service := &MemberService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

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

	_, err := service.ListMembers("org1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.CreateMemberForOrg("org1", Member{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}
