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
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
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

func exchangeCodeForToken(code, codeVerifier, clientID, redirectURI string) (*TokenResponse, error) {
	tokenURL := "https://dev-auth.hyphen.ai/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
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
		// Read and log the response body for more details
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("failed to exchange code for token: %s", bodyString)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}
func StartOAuthServer() {
	clientID := "8d5fb36d-2886-4c53-ab70-e6203e781fbc"
	redirectURI := "http://localhost:5001/token"
	authServerURL := "https://dev-auth.hyphen.ai/oauth2/auth"

	codeVerifier, codeChallenge, err := generatePKCE()
	if err != nil {
		fmt.Println("Error generating PKCE:", err)
		return
	}

	authURL, err := url.Parse(authServerURL)
	if err != nil {
		fmt.Println("Error parsing auth server URL:", err)
		return
	}

	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	query.Set("scope", "openid")
	query.Set("state", "random_state")

	authURL.RawQuery = query.Encode()

	fmt.Println("Visit the following URL to authenticate:")
	fmt.Println(authURL.String())

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		// fmt.Println("Received request", r)
		code := r.URL.Query().Get("code")
		fmt.Println("Code:", code)
		if code == "" {
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			return
		}

		token, err := exchangeCodeForToken(code, codeVerifier, clientID, redirectURI)
		if err != nil {
			http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Access Token: %s\n", token.AccessToken)
		fmt.Fprintf(w, "Refresh Token: %s\n", token.RefreshToken)
		fmt.Fprintf(w, "ID Token: %s\n", token.IDToken)
	})

	fmt.Println("Starting local server on http://localhost:5001")
	http.ListenAndServe(":5001", nil)
}
