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
	if strings.ToLower(os.Getenv("HYPHEN_Local")) == "true" {
		return "http://localhost:4000"
	}
	return "https://api.hyphen.ai"
}

func GetBaseHorizonUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "https://dev-horizon.hyphen.ai"
	}
	if strings.ToLower(os.Getenv("HYPHEN_Local")) == "true" {
		return "http://localhost:3333"
	}
	return "https://toggle.hyphen.cloud"
}

func GetBaseWebsocketUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "wss://dev-api.hyphen.ai"
	}
	if strings.ToLower(os.Getenv("HYPHEN_Local")) == "true" {
		return "ws://localhost:4000"
	}
	return "wss://api.hyphen.ai"
}

func GetBaseAppUrl() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "https://dev-app.hyphen.ai"
	}
	if strings.ToLower(os.Getenv("HYPHEN_Local")) == "true" {
		return "http://localhost:3000"
	}
	return "https://app.hyphen.ai"
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
		return "https://dev-vinz.hyphen.ai"
		//return "http://localhost:3113"
	}
	return "https://vinz.hyphen.ai"
}
