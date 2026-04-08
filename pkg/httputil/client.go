package httputil

import (
	"net/http"
	"strings"
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
	if req == nil {
		return nil, errors.New("Request is required")
	}
	if req.URL == nil {
		return nil, errors.New("Request URL is required")
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}

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
		return nil, errors.Wrapf(err, "Request failed: %s", describeRequest(req))
	}

	return resp, nil
}

func describeRequest(req *http.Request) string {
	parts := make([]string, 0, 2)

	if method := strings.TrimSpace(req.Method); method != "" {
		parts = append(parts, method)
	}
	if req.URL != nil {
		parts = append(parts, req.URL.String())
	}
	if len(parts) == 0 {
		return "request"
	}

	return strings.Join(parts, " ")
}
