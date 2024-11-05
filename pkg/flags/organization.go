package flags

import (
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetOrganizationID() (string, error) {
	if OrganizationFlag != "" {
		return OrganizationFlag, nil
	}

	m, err := manifest.RestoreConfig()
	if err != nil {
		return "", err
	}

	if m.OrganizationId == "" {
		return "", errors.New("No organization ID provided and no default found in manifest")
	}

	if !isGlobalOrgIdSameAsLocal() {
		cprint.Warning("The app organization ID is different from the global organization ID. This could lead to unexpected behavior.", VerboseFlag)
	}

	return m.OrganizationId, nil
}

func isGlobalOrgIdSameAsLocal() bool {
	ml, err := manifest.RestoreLocalConfig()
	if err != nil {
		return false
	}

	mg, err := manifest.RestoreGlobalConfig()
	if err != nil {
		return false
	}

	if ml.OrganizationId != mg.OrganizationId {
		return false
	}

	return true
}
