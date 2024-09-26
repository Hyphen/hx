package flags

import (
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetOrganizationID() (string, error) {
	if OrganizationFlag != "" {
		return OrganizationFlag, nil
	}

	if OrgFlag != "" {
		return OrgFlag, nil
	}

	manifest, err := manifest.Restore()
	if err != nil {
		return "", err
	}

	if manifest.OrganizationId == "" {
		return "", errors.New("No organization ID provided and no default found in manifest")
	}

	return manifest.OrganizationId, nil
}
