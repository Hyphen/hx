package user

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type UserServicer interface {
	GetExecutionContext() (ExecutionContext, error)
}

type UserService struct {
	baseUrl string
	client  httputil.Client
}

func NewService() UserServicer {
	baseUrl := apiconf.GetBaseApixUrl()
	return &UserService{
		baseUrl: baseUrl,
		client:  httputil.NewHyphenHTTPClient(),
	}
}

func (us *UserService) GetExecutionContext() (ExecutionContext, error) {
	req, err := http.NewRequest("GET", us.baseUrl+"/api/execution-context/", nil)
	if err != nil {
		return ExecutionContext{}, errors.Wrap(err, "Failed to prepare the request. Please try again later.")
	}

	resp, err := us.client.Do(req)
	if err != nil {
		return ExecutionContext{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ExecutionContext{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExecutionContext{}, errors.Wrap(err, "Failed to read the server response. Please try again later.")
	}

	var executionContext ExecutionContext
	err = json.Unmarshal(body, &executionContext)
	if err != nil {
		return ExecutionContext{}, errors.Wrap(err, "Failed to process the server response. Please try again later.")
	}

	return executionContext, nil
}
