package httputil

import (
	"net/http"
	"time"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type HyphenClient struct {
	client       *http.Client
	oauthService oauth.OAuthServicer
}

func NewHyphenHTTPClient() *HyphenClient {
	return &HyphenClient{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		oauthService: oauth.DefaultOAuthService(),
	}
}

func (hc *HyphenClient) Do(req *http.Request) (*http.Response, error) {
	apiKey := flags.GetAPIKey()
	if apiKey == nil {
		token, err := hc.oauthService.GetValidToken()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to authenticate. Please authenticate with `auth` and try again.")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.Header.Set("x-api-key", *apiKey)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}

	return resp, nil
}
