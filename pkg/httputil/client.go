package httputil

import (
	"net/http"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type HyphenClient struct {
	client       *http.Client
	oauthService oauth.OAuthServicer
}

func NewHyphenHTTPClient() *HyphenClient {
	return NewHyphenHTTPClientWithTimeout(time.Second * 30)
}

func NewHyphenHTTPClientWithTimeout(timeout time.Duration) *HyphenClient {
	return &HyphenClient{
		client: &http.Client{
			Timeout: timeout,
		},
		oauthService: oauth.DefaultOAuthService(),
	}
}

func (hc *HyphenClient) Do(req *http.Request) (*http.Response, error) {
	config, err := config.RestoreConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load .hx")
	}

	if config.HyphenAPIKey != nil {
		req.Header.Set("x-api-key", *config.HyphenAPIKey)
	} else {
		token, err := hc.oauthService.GetValidToken()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to authenticate. Please authenticate with `hx auth` and try again.")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Request failed: %s %s: %v", req.Method, req.URL.String(), err)
	}

	return resp, nil
}
