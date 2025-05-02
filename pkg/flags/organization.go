package flags

import (
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
)

func GetOrganizationID() (string, error) {
	if OrganizationFlag != "" {
		return OrganizationFlag, nil
	}

	m, err := config.RestoreConfig()
	if err != nil {
		return "", err
	}

	if m.OrganizationId == "" {
		return "", errors.New("No organization ID provided and no default found in manifest")
	}

	if !isGlobalOrgIdSameAsLocal() {
		cprint.Warning("The app organization ID is different from the global organization ID. This could lead to unexpected behavior.")
	}

	return m.OrganizationId, nil
}

func isGlobalOrgIdSameAsLocal() bool {
	ml, _ := config.RestoreLocalConfig() // we can skip the error, if its new project will no have ml
	if ml.OrganizationId == "" {
		return true //if we dont have ml return true bc is new project
	}

	mg, _ := config.RestoreGlobalConfig() // we can skip this error check, bc whe check the error in the line  16

	if ml.OrganizationId != mg.OrganizationId {
		return false
	}

	return true
}
