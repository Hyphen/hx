package user

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type UserServicer interface {
	GetExecutionContext() (models.ExecutionContext, error)
}

type UserService struct {
	baseUrl string
	client  httputil.Client
}

func ErrorIfNotAuthenticated() error {
	// Try a simple request to force tokens refresh and validate we can get a response
	_, err := NewService().GetExecutionContext()
	if errors.Is(err, errors.ErrUnauthorized) || errors.Is(err, errors.ErrForbidden) {
		return errors.New("You are not authenticated. Please run `hx auth` and try again")
	} else if err != nil {
		return err
	}

	return nil
}

func NewService() UserServicer {
	baseUrl := apiconf.GetBaseApixUrl()
	return &UserService{
		baseUrl: baseUrl,
		client:  httputil.NewHyphenHTTPClient(),
	}
}

func (us *UserService) GetExecutionContext() (models.ExecutionContext, error) {
	req, err := http.NewRequest("GET", us.baseUrl+"/api/execution-context/", nil)
	if err != nil {
		return models.ExecutionContext{}, errors.Wrap(err, "Failed to prepare the request. Please try again later.")
	}

	resp, err := us.client.Do(req)
	if err != nil {
		return models.ExecutionContext{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ExecutionContext{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ExecutionContext{}, errors.Wrap(err, "Failed to read the server response. Please try again later.")
	}

	var executionContext models.ExecutionContext
	err = json.Unmarshal(body, &executionContext)
	if err != nil {
		return models.ExecutionContext{}, errors.Wrap(err, "Failed to process the server response. Please try again later.")
	}

	return executionContext, nil
}
