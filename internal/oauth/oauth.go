package oauth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/timeprovider"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
)

var (
	redirectURI = "http://localhost:5001/token"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiryTime   int64  `json:"-"`
}

var errorMessages = map[int]string{
	http.StatusBadRequest:          "Bad request. Please check the request parameters and try again.",
	http.StatusUnauthorized:        "Unauthorized. Please authenticate with `auth` and try again.",
	http.StatusInternalServerError: "Internal server error. Please try again later.",
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type BrowserOpener func(string) error

type OAuthServicer interface {
	IsTokenExpired(expiryTime int64) bool
	RefreshToken(refreshToken string) (*TokenResponse, error)
	GetValidToken() (string, error)
}

// Ensure OAuthService implements OAuthServiceInterface
var _ OAuthServicer = (*OAuthService)(nil)

type OAuthService struct {
	baseUrl       string
	clientID      string
	httpClient    HTTPClient
	timeProvider  timeprovider.TimeProvider
	browserOpener BrowserOpener
}

func DefaultOAuthService() *OAuthService {
	return NewOAuthService(&http.Client{}, timeprovider.DefaultTimeProvider(), openBrowser)
}

func NewOAuthService(httpClient HTTPClient, timeProvider timeprovider.TimeProvider, browserOpener BrowserOpener) *OAuthService {
	baseUrl := apiconf.GetBaseAuthUrl()
	clientID := apiconf.GetAuthClientID()

	return &OAuthService{
		baseUrl,
		clientID,
		httpClient,
		timeProvider,
		browserOpener,
	}
}

func (s *OAuthService) generatePKCE() (string, string, error) {
	codeVerifier := make([]byte, 32)
	_, err := rand.Read(codeVerifier)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to generate PKCE code verifier")
	}

	codeVerifierStr := base64.RawURLEncoding.EncodeToString(codeVerifier)

	hash := sha256.New()
	hash.Write([]byte(codeVerifierStr))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

	return codeVerifierStr, codeChallenge, nil
}

func (s *OAuthService) exchangeCodeForToken(code, codeVerifier string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/token", s.baseUrl)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")

	data.Set("client_id", s.clientID)
	data.Set("code_verifier", codeVerifier)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create token exchange request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send token exchange request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, errors.New(fmt.Sprintf("Failed to exchange code for token: %s", bodyString))
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, errors.Wrap(err, "Failed to decode token response")
	}

	tokenResponse.ExpiryTime = s.timeProvider.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()

	return &tokenResponse, nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	case "darwin":
		cmd = "open"
	default:
		return errors.New("Unsupported platform for opening browser")
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func (s *OAuthService) StartOAuthServer() (*TokenResponse, error) {
	authServerURL := fmt.Sprintf("%s/oauth2/auth", s.baseUrl)

	codeVerifier, codeChallenge, err := s.generatePKCE()
	if err != nil {
		return nil, err
	}

	authURL, err := url.Parse(authServerURL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse authentication server URL")
	}

	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", s.clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	query.Set("scope", "openid offline_access")

	// Generate a random base64-encoded string to be state. It's unused but required.
	state := make([]byte, 32)
	_, _ = rand.Read(state)
	query.Set("state", base64.RawURLEncoding.EncodeToString(state))

	authURL.RawQuery = query.Encode()

	fmt.Println("Visit the following URL to authenticate:")
	fmt.Println(authURL.String())

	_ = s.browserOpener(authURL.String())

	tokenChan := make(chan *TokenResponse)
	errorChan := make(chan error)
	defer close(tokenChan)
	defer close(errorChan)

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, errorMessages[http.StatusBadRequest], http.StatusBadRequest)
			errorChan <- errors.New("Authorization code not found")
			return
		}

		token, err := s.exchangeCodeForToken(code, codeVerifier)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if respErr, ok := err.(*url.Error); ok && respErr.Timeout() {
				statusCode = http.StatusRequestTimeout
			}
			errorMessage, found := errorMessages[statusCode]
			if !found {
				errorMessage = "An unexpected error occurred. Please try again later."
			}
			http.Error(w, errorMessage, statusCode)
			errorChan <- err
			return
		}

		http.Redirect(w, r, "https://hyphen.ai/cli?authenticated=true", http.StatusTemporaryRedirect)

		tokenChan <- token
	})

	server := &http.Server{Addr: ":5001"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- errors.Wrap(err, "Failed to start local server for OAuth flow")
		}
	}()

	select {
	case token := <-tokenChan:
		server.Close()
		return token, nil
	case err := <-errorChan:
		server.Close()
		return nil, err
	}
}

func (s *OAuthService) IsTokenExpired(expiryTime int64) bool {
	return s.timeProvider.Now().Unix() > expiryTime
}

func (s *OAuthService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/token", s.baseUrl)

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", s.clientID)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create refresh token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send refresh token request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, errors.New(fmt.Sprintf("Failed to refresh token: %s", bodyString))
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, errors.Wrap(err, "Failed to decode refresh token response")
	}

	tokenResponse.ExpiryTime = s.timeProvider.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()

	return &tokenResponse, nil
}

func (s *OAuthService) GetValidToken() (string, error) {
	mc, err := config.RestoreConfig()
	if err != nil {
		return "", err
	}

	if mc.ExpiryTime == nil || mc.HyphenRefreshToken == nil {
		return "", errors.New("You must authenticate. Run `hx auth` and try again.")
	}

	if s.IsTokenExpired(*mc.ExpiryTime) {
		tokenResponse, err := s.RefreshToken(*mc.HyphenRefreshToken)
		if err != nil {
			return "", errors.Wrap(err, "Failed to refresh token")
		}
		mc.HyphenAccessToken = &tokenResponse.AccessToken
		mc.HyphenRefreshToken = &tokenResponse.RefreshToken
		mc.HypenIDToken = &tokenResponse.IDToken
		mc.ExpiryTime = &tokenResponse.ExpiryTime
		err = config.UpsertGlobalConfig(mc)
		if err != nil {
			return "", errors.Wrap(err, "Failed to save refreshed credentials")
		}
		return tokenResponse.AccessToken, nil
	}
	return *mc.HyphenAccessToken, nil
}
