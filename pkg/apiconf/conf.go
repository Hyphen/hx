package apiconf

import "os"

func GetBaseApixUrl() string {
	baseUrl := os.Getenv("HYPHEN_CUSTOM_APIX")
	if baseUrl == "" {
		baseUrl = "https://api.hyphen.ai"
	}
	return baseUrl
}

func GetBaseAuthUrl() string {
	baseUrl := os.Getenv("HYPHEN_CUSTOM_AUTH")
	if baseUrl == "" {
		baseUrl = "https://auth.hyphen.ai"
	}
	return baseUrl
}
