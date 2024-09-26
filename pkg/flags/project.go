package flags

import (
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetProjectID() (string, error) {
	if ProjectFlag != "" {
		return ProjectFlag, nil
	}

	manifest, err := manifest.Restore()
	if err != nil {
		return "", err
	}

	if manifest.ProjectId == nil {
		return "", errors.New("No Project ID provided and no default found in manifest")
	}

	return *manifest.ProjectId, nil
}
