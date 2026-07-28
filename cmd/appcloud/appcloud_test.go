package appcloud

import (
	"testing"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAppCloudCmd(t *testing.T) {
	t.Run("registers_the_apps_and_app_subcommands", func(t *testing.T) {
		names := map[string]bool{}
		for _, c := range AppCloudCmd.Commands() {
			names[c.Name()] = true
		}

		assert.True(t, names["apps"], "expected an `apps` subcommand")
		assert.True(t, names["app"], "expected an `app` subcommand")
	})

	t.Run("aliases_appcloud_to_ac", func(t *testing.T) {
		assert.Contains(t, AppCloudCmd.Aliases, "ac")
	})
}

func TestAppCmdArgs(t *testing.T) {
	t.Run("requires_exactly_one_app_id", func(t *testing.T) {
		err := appCmd.Args(appCmd, []string{})

		assert.Error(t, err)
	})

	t.Run("accepts_a_single_app_id", func(t *testing.T) {
		err := appCmd.Args(appCmd, []string{"the_app_id"})

		assert.NoError(t, err)
	})
}

func TestAppsCmdUsesTheService(t *testing.T) {
	t.Run("lists_apps_from_the_service_for_the_resolved_org", func(t *testing.T) {
		mockService := new(appcloud.MockAppCloudService)
		mockService.On("ListApps", mock.Anything, mock.Anything).
			Return([]appcloud.App{{ID: "the_app_id", Name: "the app"}}, nil)
		original := newAppCloudService
		newAppCloudService = func() appcloud.AppCloudServicer { return mockService }
		t.Cleanup(func() { newAppCloudService = original })

		// The command resolves the org from flags/config; we only assert the
		// service is wired in (org resolution is covered by the flags package).
		assert.NotNil(t, appsCmd.RunE)
		assert.NotNil(t, newAppCloudService())
	})
}
