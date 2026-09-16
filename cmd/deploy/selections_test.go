package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployArguments(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "index.html")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0600))
	for _, test := range []struct {
		name, kind    string
		args          []string
		noBuild       bool
		dockerfile    string
		id, directory string
		fail          bool
	}{
		{name: "default docker", kind: "docker"},
		{name: "docker id", kind: "docker", args: []string{"dply_test"}, id: "dply_test"},
		{name: "static directory", kind: "static", args: []string{dir}, directory: dir},
		{name: "static id directory", kind: "static", args: []string{"dply_test", dir}, id: "dply_test", directory: dir},
		{name: "static no build", kind: "static", args: []string{"dply_test"}, noBuild: true, id: "dply_test"},
		{name: "static default no build", kind: "static", noBuild: true},
		{name: "missing directory", kind: "static", fail: true},
		{name: "file not directory", kind: "static", args: []string{file}, fail: true},
		{name: "static dockerfile", kind: "static", args: []string{dir}, dockerfile: "Dockerfile", fail: true},
		{name: "invalid type", kind: "tar", noBuild: true, fail: true},
		{name: "docker extra directory", kind: "docker", args: []string{"dply_test", dir}, fail: true},
		{name: "no build extra directory", kind: "static", args: []string{"dply_test", dir}, noBuild: true, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			id, directory, err := deployArguments(test.args, test.kind, test.noBuild, test.dockerfile)
			if test.fail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.id, id)
			assert.Equal(t, test.directory, directory)
		})
	}
}

func mixedDeployment() models.Deployment {
	return models.Deployment{
		Apps:  []models.DeploymentApp{{App: models.AppReference{ID: "app_api", AlternateID: "api"}}},
		Sites: []models.DeploymentSite{{App: models.AppReference{ID: "app_web", AlternateID: "website"}}, {App: models.AppReference{ID: "app_docs", AlternateID: "docs"}}},
	}
}

func TestSeparateAppAndSiteSelectors(t *testing.T) {
	selections, err := parseSelections("api:lastDeployed", "website:abld_static, docs:latestPreview")
	require.NoError(t, err)
	sources, err := deploymentSources(mixedDeployment(), selections)
	require.NoError(t, err)
	require.Len(t, sources, 3)
	assert.Equal(t, "app_api", sources[0].AppId)
	assert.Equal(t, "lastDeployed", sources[0].Build)
	assert.Equal(t, "app_web", sources[1].AppId)
	assert.Equal(t, "abld_static", sources[1].BuildId)
	assert.Empty(t, sources[1].Build)
	assert.Equal(t, "latestPreview", sources[2].Build)
	// --no-build consumes the same sources, without overwriting explicit selectors.
	for _, value := range []string{"latest", "lastDeployed", "latestPreview", "abld_123"} {
		selected, err := parseSelections("", "website:"+value)
		require.NoError(t, err)
		sources, err := deploymentSources(mixedDeployment(), selected)
		require.NoError(t, err)
		assert.Equal(t, value, sources[0].Build+sources[0].BuildId)
	}
}

func TestSelectionsRejectConflicts(t *testing.T) {
	for _, test := range []struct{ apps, sites string }{
		{"api:banana", ""}, {"", ":"}, {"api,api", ""}, {"api", "api"}, {"", "website:abld_"}, {"", ","},
	} {
		_, err := parseSelections(test.apps, test.sites)
		require.Error(t, err)
	}
	for _, selected := range [][]selection{
		{{ID: "api", Kind: "static"}}, {{ID: "website", Kind: "docker"}}, {{ID: "missing", Kind: "static"}},
		{{ID: "website", Kind: "static"}, {ID: "app_web", Kind: "static"}},
	} {
		_, err := deploymentSources(mixedDeployment(), selected)
		require.Error(t, err)
	}
	// Wrong membership must fail before a service is touched.
	_, err := ensureSelections(nil, "org", mixedDeployment(), []selection{{ID: "website", Kind: "docker"}})
	require.ErrorContains(t, err, "already configured as static")
}

func TestBuildTypeOnlySelectsTheLocalBuild(t *testing.T) {
	id, alias := "app_web", "website"
	cfg := config.Config{AppId: &id, AppAlternateId: &alias}
	selected, err := parseSelections("api:latest", "website,docs:lastDeployed")
	require.NoError(t, err)
	assert.True(t, selectsLocalBuild(selected, cfg, "static"))
	assert.False(t, selectsLocalBuild(selected, cfg, "docker"))
	selected[1].BuildSpec = "latest"
	assert.False(t, selectsLocalBuild(selected, cfg, "static"))
}

func TestDefaultSelectionIncludesSitesAndApps(t *testing.T) {
	deployment := mixedDeployment()
	sources, err := deploymentSources(deployment, allSelections(deployment))
	require.NoError(t, err)
	require.Len(t, sources, 3)
	for _, source := range sources {
		assert.Equal(t, "latest", source.Build)
	}
	assert.Equal(t, "docker", DeployCmd.Flags().Lookup("type").DefValue)
	assert.NotNil(t, DeployCmd.Flags().Lookup("sites"))
	assert.NotNil(t, DeployCmd.Flags().Lookup("apps"))
}
