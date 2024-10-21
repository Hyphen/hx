package flags

import (
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetApplicationID() (string, error) {
	if ApplicationFlag != "" {
		return ApplicationFlag, nil
	}

	manifest, err := manifest.Restore()
	if err != nil {
		return "", err
	}

	if manifest.AppId == nil {
		return "", errors.New("No app ID provided and no default found in manifest")
	}

	return *manifest.AppId, nil
}
