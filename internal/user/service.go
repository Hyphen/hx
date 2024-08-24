package user

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
)

type UserServicer interface {
	GetUserInformation() (UserInfo, error)
}

type UserService struct {
	baseUrl      string
	oauthService oauth.OAuthServiceInterface
}

func NewService() UserServicer {
	baseUrl := "https://dev-api.hyphen.ai"
	if customAPI := os.Getenv("HYPHEN_CUSTOM_APIX"); customAPI != "" {
		baseUrl = customAPI
	}
	return &UserService{
		baseUrl,
		oauth.DefaultOAuthService(),
	}
}

func (us *UserService) GetUserInformation() (UserInfo, error) {
	token, err := us.oauthService.GetValidToken()
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", us.baseUrl+"/api/me/", nil)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to prepare the request. Please try again later.")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return UserInfo{}, errors.Wrap(err, "Failed to connect to the server. Please check your internet connection and try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserInfo{}, errors.New(fmt.Sprintf("Server returned an error (status code %d). Please try again later.", resp.StatusCode))
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
