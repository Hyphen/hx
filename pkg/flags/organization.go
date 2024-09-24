package flags

import (
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetOrganizationID() (string, error) {
	if OrgFlag != "" {
		return OrgFlag, nil
	}

	manifest, err := manifest.Restore()
	if err != nil {
		return "", errors.Wrap(err, "Failed to load credentials")
	}

	if manifest.OrganisationId == "" {
		return "", errors.New("No organization ID provided and no default found in manifest")
	}

	return manifest.OrganisationId, nil
}
