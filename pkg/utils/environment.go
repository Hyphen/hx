package utils

import (
	"fmt"

	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/pkg/errors"
)

// If no environment is specified, it will return an empty string.
func GetEnvronmentID() (string, error) {
	if EnvironmentFlag != "" {
		envName, err := env.GetEnvName(EnvironmentFlag)
		if err != nil {
			return "", errors.Wrap(err, fmt.Sprintf("environment '%s' is not valid", EnvironmentFlag))
		}
		return envName, nil
	}

	return "", nil

}
