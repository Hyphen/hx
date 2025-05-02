package flags

import (
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetProjectID() (string, error) {
	if ProjectFlag != "" {
		return ProjectFlag, nil
	}

	config, err := config.RestoreConfig()
	if err != nil {
		return "", err
	}

	if config.ProjectId == nil {
		return "", errors.New("No Project ID provided and no default found in .hx file")
	}

	return *config.ProjectId, nil
}
