package projects_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/projects"
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

var organizationID = "org_123"

func TestListProjects(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	ps := projects.NewTestService(organizationID, mockHTTPClient)
	mockResponseBody := `{ "data": [{"id": "1", "name": "Project 1"}] }`
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(mockResponseBody)),
	}
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	projects, err := ps.ListProjects()
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, "1", *projects[0].ID)
}

func TestGetProject(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	ps := projects.NewTestService(organizationID, mockHTTPClient)
	mockResponseBody := `{"id": "1", "name": "Project 1"}`
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(mockResponseBody)),
	}
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	project, err := ps.GetProject("1")
	assert.NoError(t, err)
	assert.Equal(t, "1", *project.ID)
}

func TestCreateProject(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	ps := projects.NewTestService(organizationID, mockHTTPClient)
	mockResponseBody := `{"id": "1", "name": "Project 1"}`
	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(mockResponseBody)),
	}
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	newProject := models.Project{
		Name: "Project 1",
	}

	createdProject, err := ps.CreateProject(newProject)
	assert.NoError(t, err)
	assert.Equal(t, "1", *createdProject.ID)
}
