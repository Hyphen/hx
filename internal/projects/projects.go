package projects

import (
	"regexp"
	"strings"

	"github.com/Hyphen/cli/pkg/errors"
)

type Project struct {
	ID          *string `json:"id,omitempty"`
	AlternateID string  `json:"alternateId"`
	Name        string  `json:"name"`
	IsMonorepo  bool    `json:"isMonorepo"`
}

func CheckProjectId(appId string) error {
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(appId) {
		suggested := strings.ToLower(appId)
		suggested = strings.ReplaceAll(suggested, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return errors.Wrapf(
			errors.New("invalid project ID"),
			"You are using unpermitted characters. A valid Project ID can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid ID: %s",
			suggested,
		)
	}

	return nil
}
