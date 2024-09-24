package flags

import (
	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetOrganizationID() (string, error) {
	if OrgFlag != "" {
		return OrgFlag, nil
	}

	credentials, err := config.LoadCredentials()
	if err != nil {
		return "", errors.Wrap(err, "Failed to load credentials")
	}

	if credentials.Default.OrganizationId == "" {
		return "", errors.New("No organization ID provided and no default found in credentials")
	}

	return credentials.Default.OrganizationId, nil
}
