package flags

import (
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetApplicationID() (string, error) {
	if ApplicationFlag != "" {
		return ApplicationFlag, nil
	}

	manifest, err := config.RestoreConfig()
	if err != nil {
		return "", err
	}

	if manifest.AppId == nil {
		return "", errors.New("No app ID provided and no default found in manifest")
	}

	return *manifest.AppId, nil
}
