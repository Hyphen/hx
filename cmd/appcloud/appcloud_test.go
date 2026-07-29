package appcloud

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func subcommandNames(c *cobra.Command) map[string]bool {
	names := map[string]bool{}
	for _, sub := range c.Commands() {
		names[sub.Name()] = true
	}
	return names
}

func TestAppCloudCmd(t *testing.T) {
	t.Run("registers_deploy_apps_app_and_metrics", func(t *testing.T) {
		names := subcommandNames(AppCloudCmd)

		for _, want := range []string{"deploy", "apps", "app", "metrics"} {
			assert.True(t, names[want], "expected an `%s` subcommand", want)
		}
	})

	t.Run("aliases_appcloud_to_ac", func(t *testing.T) {
		assert.Contains(t, AppCloudCmd.Aliases, "ac")
	})
}

func TestAppsGroup(t *testing.T) {
	t.Run("registers_the_crud_and_deploy_subcommands", func(t *testing.T) {
		names := subcommandNames(appsCmd)

		for _, want := range []string{"list", "get", "create", "delete", "deploy", "config"} {
			assert.True(t, names[want], "expected `apps %s`", want)
		}
	})
}

func TestMetricsGroup(t *testing.T) {
	t.Run("registers_http_and_errors", func(t *testing.T) {
		names := subcommandNames(metricsCmd)

		assert.True(t, names["http"])
		assert.True(t, names["errors"])
	})
}

func TestResolveAppID(t *testing.T) {
	t.Run("returns_the_explicit_id_when_given", func(t *testing.T) {
		id, err := resolveAppID("the_app_id")

		assert.NoError(t, err)
		assert.Equal(t, "the_app_id", id)
	})
}

func TestDeployCmdArgs(t *testing.T) {
	t.Run("requires_exactly_one_directory_argument", func(t *testing.T) {
		assert.Error(t, deployCmd.Args(deployCmd, []string{}))
		assert.NoError(t, deployCmd.Args(deployCmd, []string{"./dist"}))
	})
}
