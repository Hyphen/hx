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
	GetUserInformation() (UserInfo, error)
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

func (us *UserService) GetUserInformation() (UserInfo, error) {
	// TODO - this needs to switch to useing /api/execution-context instead of /me. Lots of updates to make that happen.
	req, err := http.NewRequest("GET", us.baseUrl+"/api/me/", nil)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to prepare the request. Please try again later.")
	}

	resp, err := us.client.Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserInfo{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to read the server response. Please try again later.")
	}

	var userInfo UserInfo
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to process the server response. Please try again later.")
	}

	return userInfo, nil
}
