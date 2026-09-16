package toggle

import (
	"testing"

	"github.com/Hyphen/cli/pkg/flags"
	"github.com/stretchr/testify/assert"
)

func TestProviderAlwaysUsesProductionHorizon(t *testing.T) {
	for _, test := range []struct {
		name, local, apix, horizon, dev, environment string
	}{
		{"production", "false", "false", "false", "false", "production"},
		{"dev horizon with local apix", "false", "true", "false", "true", "development"},
		{"local horizon with dev apix", "false", "false", "true", "true", "development"},
		{"local overrides both disabled", "true", "false", "false", "true", "development"},
		{"both enabled independently", "false", "true", "true", "true", "development"},
		{"local without dev environment", "true", "false", "false", "false", "production"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HYPHEN_LOCAL", test.local)
			t.Setenv("HYPHEN_LOCAL_APIX", test.apix)
			t.Setenv("HYPHEN_LOCAL_HORIZON", test.horizon)
			t.Setenv("HYPHEN_DEV", test.dev)
			previous := flags.DevFlag
			flags.DevFlag = false
			t.Cleanup(func() { flags.DevFlag = previous })
			config := providerConfig()
			assert.Equal(t, test.environment, config.Environment)
			assert.Empty(t, config.HorizonUrls, "retain the SDK's organization-specific production endpoint")
		})
	}
}
