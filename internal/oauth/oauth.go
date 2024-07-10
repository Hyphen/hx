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
	"os"
	"os/exec"
	"runtime"
	"time"
)

var (
	devBaseUrl  = "https://dev-auth.hyphen.ai"
	prodBaseUrl = "https://auth.hyphen.ai"
	clientID    = "8d5fb36d-2886-4c53-ab70-e6203e781fbc"
	redirectURI = "http://localhost:5001/token"
	secretKey   = "klUG3PV9lmeddshcKJEf5YTDXl"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"` // Token validity duration in seconds
	ExpiryTime   int64  `json:"-"`          // Unix timestamp when the token expires
}

var errorMessages = map[int]string{
	http.StatusBadRequest:          "Bad request. Please check the request parameters and try again.",
	http.StatusUnauthorized:        "Unauthorized. Please check your credentials and try again.",
	http.StatusInternalServerError: "Internal server error. Please try again later.",
}

func baseUrl() string {
	if os.Getenv("HYPHEN_CLI_ENV") == "dev" {
		return devBaseUrl
	}
	return prodBaseUrl
}

func generatePKCE() (string, string, error) {
	codeVerifier := make([]byte, 32)
	_, err := rand.Read(codeVerifier)
	if err != nil {
		return "", "", err
	}

	codeVerifierStr := base64.RawURLEncoding.EncodeToString(codeVerifier)

	hash := sha256.New()
	hash.Write([]byte(codeVerifierStr))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

	return codeVerifierStr, codeChallenge, nil
}

func exchangeCodeForToken(code, codeVerifier string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/token", baseUrl())

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", secretKey)
	data.Set("code_verifier", codeVerifier)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("failed to exchange code for token: %s", bodyString)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	// Calculate and store the expiry time
	tokenResponse.ExpiryTime = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()

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
		return fmt.Errorf("unsupported platform")
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func StartOAuthServer() (*TokenResponse, error) {
	authServerURL := fmt.Sprintf("%s/oauth2/auth", baseUrl())

	codeVerifier, codeChallenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}

	authURL, err := url.Parse(authServerURL)
	if err != nil {
		return nil, err
	}

	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	query.Set("scope", "openid offline_access")
	query.Set("state", "random_state")

	authURL.RawQuery = query.Encode()

	fmt.Println("Visit the following URL to authenticate:")
	fmt.Println(authURL.String())

	_ = openBrowser(authURL.String())

	tokenChan := make(chan *TokenResponse)
	errorChan := make(chan error)
	defer close(tokenChan)
	defer close(errorChan)

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, errorMessages[http.StatusBadRequest], http.StatusBadRequest)
			errorChan <- fmt.Errorf("authorization code not found")
			return
		}

		token, err := exchangeCodeForToken(code, codeVerifier)
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

		// Serve a completion page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Authentication Complete</title>
			</head>
			<body>
				<h1>Authentication Complete</h1>
				<h2>You can now close this window.</h2>
			</body>
			</html>
		`)

		tokenChan <- token
	})

	server := &http.Server{Addr: ":5001"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	select {
	case token := <-tokenChan:
		server.Close() // Gracefully shut down the server
		return token, nil
	case err := <-errorChan:
		server.Close() // Gracefully shut down the server
		return nil, err
	}
}

func IsTokenExpired(expiryTime int64) bool {
	return time.Now().Unix() > expiryTime
}

func RefreshToken(refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/token", baseUrl())

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", secretKey)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("failed to refresh token: %s", bodyString)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	// Calculate and store the new expiry time
	tokenResponse.ExpiryTime = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()

	return &tokenResponse, nil
}
