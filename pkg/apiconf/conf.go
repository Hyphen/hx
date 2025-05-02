package apiconf

import (
	"os"
	"strings"

	"github.com/Hyphen/cli/pkg/flags"
)

func GetBaseApixUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "https://dev-api.hyphen.ai"
	}
	return "https://api.hyphen.ai"
}

func GetBaseAuthUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "https://dev-auth.hyphen.ai"
	}
	return "https://auth.hyphen.ai"
}

func GetAuthClientID() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "8d5fb36d-2886-4c53-ab70-e6203e781fbc"
	}
	return "e6315ab1-5847-4c75-a003-65b5ed374dd1"
}

func GetBaseVinzUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		//return "https://dev-vinz.hyphen.ai"
		return "http://localhost:3113"
	}
	return "https://vinz.hyphen.ai"
}
