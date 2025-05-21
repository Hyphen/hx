package httputil

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/gorilla/websocket"
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
	config, err := config.RestoreConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load .hx")
	}

	if config.HyphenAPIKey != nil {
		req.Header.Set("x-api-key", *config.HyphenAPIKey)
	} else {
		token, err := hc.oauthService.GetValidToken()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to authenticate. Please authenticate with `auth` and try again.")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}

	return resp, nil
}

func (hc *HyphenClient) GetWebsocketConnection(websocketUrl string) (*websocket.Conn, error) {

	headers := http.Header{}

	config, err := config.RestoreConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load .hx")
	}

	if config.HyphenAPIKey != nil {
		headers.Set("x-api-key", *config.HyphenAPIKey)
	} else {
		token, err := hc.oauthService.GetValidToken()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to authenticate. Please authenticate with `auth` and try again.")
		}
		headers.Set("Authorization", "Bearer "+token)
	}

	ws, resp, err := websocket.DefaultDialer.Dial(websocketUrl, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 101 {
		return nil, fmt.Errorf("error connecting to websocket: %s", resp.Status)
	}
	return ws, nil
}
