package app

import (
	"regexp"
	"strings"

	"github.com/Hyphen/cli/pkg/errors"
)

type App struct {
	ID           string       `json:"id"`
	AlternateId  string       `json:"alternateId"`
	Name         string       `json:"name"`
	Organization Organization `json:"organization"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func CheckAppId(appId string) error {
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(appId) {
		suggested := strings.ToLower(appId)
		suggested = strings.ReplaceAll(suggested, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return errors.Wrapf(
			errors.New("invalid project ID"),
			"You are using unpermitted characters. A valid App ID can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid ID: %s",
			suggested,
		)
	}

	return nil
}
