package deploy

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/build"
	"github.com/Hyphen/cli/internal/config"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/internal/models"
)

func deployArguments(args []string, buildType string, noBuild bool, dockerfile string) (deploymentID, directory string, err error) {
	if buildType == "static" && !noBuild {
		if len(args) == 0 || len(args) > 2 {
			return "", "", fmt.Errorf("usage: deploy [deploymentId] --type static <site-directory>")
		}
		directory = args[len(args)-1]
		args = args[:len(args)-1]
	} else if len(args) > 1 {
		return "", "", fmt.Errorf("expected at most one deployment ID; a directory is only used with --type static without --no-build")
	}
	if noBuild && buildType == "static" {
		if dockerfile != "" {
			return "", "", fmt.Errorf("--dockerfile cannot be used with --type static")
		}
	} else if err := build.ValidateInput(buildType, directory, dockerfile); err != nil {
		return "", "", err
	}
	if len(args) == 1 {
		deploymentID = args[0]
	}
	return deploymentID, directory, nil
}

type selection struct {
	ID        string
	BuildSpec string
	Kind      string
}

func parseSelections(apps, sites string) ([]selection, error) {
	var selections []selection
	seen := map[string]bool{}
	for _, group := range []struct{ value, kind, flag string }{{apps, "docker", "apps"}, {sites, "static", "sites"}} {
		if group.value == "" {
			continue
		}
		for _, entry := range strings.Split(group.value, ",") {
			id, spec, _ := strings.Cut(strings.TrimSpace(entry), ":")
			id, spec = strings.TrimSpace(id), strings.TrimSpace(spec)
			if id == "" {
				return nil, fmt.Errorf("--%s contains an empty app ID", group.flag)
			}
			if seen[id] {
				return nil, fmt.Errorf("app %q is selected more than once in --apps/--sites", id)
			}
			seen[id] = true
			if spec != "" && spec != "latest" && spec != "lastDeployed" && spec != "latestPreview" && !(strings.HasPrefix(spec, "abld_") && len(spec) > 5 && !strings.ContainsAny(spec, ": \t\n")) {
				return nil, fmt.Errorf("unknown build selector %q: expected latest, lastDeployed, latestPreview, or abld_...", spec)
			}
			selections = append(selections, selection{ID: id, BuildSpec: spec, Kind: group.kind})
		}
	}
	return selections, nil
}

func deploymentMember(identifier string, deployment models.Deployment) (models.AppReference, string) {
	for _, app := range deployment.Apps {
		if app.App.ID == identifier || app.App.AlternateID == identifier {
			return app.App, "docker"
		}
	}
	for _, site := range deployment.Sites {
		if site.App.ID == identifier || site.App.AlternateID == identifier {
			return site.App, "static"
		}
	}
	return models.AppReference{}, ""
}

func ensureSelections(service *Deployment.DeploymentService, orgID string, deployment models.Deployment, selections []selection) (models.Deployment, error) {
	var missingApps, missingSites []string
	for _, selected := range selections {
		_, kind := deploymentMember(selected.ID, deployment)
		if kind != "" && kind != selected.Kind {
			return deployment, fmt.Errorf("app %q is already configured as %s, not %s; update deployment settings explicitly", selected.ID, kind, selected.Kind)
		}
		if kind == "" {
			if selected.Kind == "static" {
				missingSites = append(missingSites, selected.ID)
			} else {
				missingApps = append(missingApps, selected.ID)
			}
		}
	}
	// Resolve the target before either PATCH so missing registry setup cannot
	// leave an otherwise mixed selection partially updated.
	integrationID := ""
	if len(missingSites) > 0 {
		registry, err := build.NewService().FindSiteRegistry(orgID, deployment.Project.ID)
		if err != nil {
			return deployment, err
		}
		integrationID = registry.OrganizationIntegration.Id
	}
	if len(missingApps) > 0 {
		printer.Print(fmt.Sprintf("Adding apps %v with default settings", missingApps))
		updated, err := service.AddAppsToDeployment(orgID, deployment.ID, missingApps)
		if err != nil {
			return deployment, err
		}
		deployment = *updated
	}
	if len(missingSites) > 0 {
		printer.Print(fmt.Sprintf("Adding sites %v with HyphenCloud target", missingSites))
		updated, err := service.AddSitesToDeployment(orgID, deployment.ID, missingSites, integrationID)
		if err != nil {
			return deployment, err
		}
		deployment = *updated
	}
	return deployment, nil
}

func allSelections(deployment models.Deployment) []selection {
	var selections []selection
	for _, app := range deployment.Apps {
		selections = append(selections, selection{ID: app.App.ID, Kind: "docker"})
	}
	for _, site := range deployment.Sites {
		selections = append(selections, selection{ID: site.App.ID, Kind: "static"})
	}
	return selections
}

func selectsLocalBuild(selections []selection, cfg config.Config, buildType string) bool {
	for _, selected := range selections {
		if selected.Kind == buildType && selected.BuildSpec == "" && matchesHxApp(selected.ID, cfg) {
			return true
		}
	}
	return false
}

func deploymentSources(deployment models.Deployment, selections []selection) ([]Deployment.AppSources, error) {
	sources := make([]Deployment.AppSources, 0, len(selections))
	seen := map[string]bool{}
	for _, selected := range selections {
		app, kind := deploymentMember(selected.ID, deployment)
		if kind != selected.Kind {
			return nil, fmt.Errorf("%s app %q not found in deployment", selected.Kind, selected.ID)
		}
		if seen[app.ID] {
			return nil, fmt.Errorf("app %q selected more than once", app.ID)
		}
		seen[app.ID] = true
		source := Deployment.AppSources{AppId: app.ID, Build: selected.BuildSpec}
		if source.Build == "" {
			source.Build = "latest"
		}
		if strings.HasPrefix(source.Build, "abld_") {
			source.BuildId, source.Build = source.Build, ""
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no apps or sites selected for deployment")
	}
	return sources, nil
}
