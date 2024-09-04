package user

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/conf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type UserServicer interface {
	GetUserInformation() (UserInfo, error)
}

type UserService struct {
	baseUrl      string
	oauthService oauth.OAuthServicer
	client       httputil.Client
}

func NewService() UserServicer {
	baseUrl := conf.GetBaseApixUrl()
	return &UserService{
		baseUrl:      baseUrl,
		oauthService: oauth.DefaultOAuthService(),
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (us *UserService) GetUserInformation() (UserInfo, error) {
	token, err := us.oauthService.GetValidToken()
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	req, err := http.NewRequest("GET", us.baseUrl+"/api/me/", nil)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to prepare the request. Please try again later.")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := us.client.Do(req)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to connect to the server. Please check your internet connection and try again.")
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
