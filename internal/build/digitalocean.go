package build

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/dockerutil"
)

var digitalOceanDestination = regexp.MustCompile(`^registry\.digitalocean\.com/[a-z0-9][a-z0-9-]{1,61}[a-z0-9]/(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)+[a-z0-9]+(?:[._-][a-z0-9]+)*:[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$`)

func isDigitalOceanRegistry(uri string) bool {
	return strings.HasPrefix(uri, "registry.digitalocean.com/")
}

func digitalOceanImageURI(registryURL, app, tag string) (string, error) {
	uri := registryURL + "/" + app + ":" + tag
	if !digitalOceanDestination.MatchString(uri) {
		return "", errors.New("invalid DigitalOcean image destination")
	}
	return uri, nil
}

func (bs *BuildService) pushDigitalOcean(ctx context.Context, configuration config.Config, registry models.ContainerRegistry, image, tag string) (string, error) {
	if configuration.ProjectId == nil || configuration.AppAlternateId == nil {
		return "", errors.New("DigitalOcean uploads require a project and app")
	}
	destination, err := digitalOceanImageURI(registry.Url, *configuration.AppAlternateId, tag)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		registries, err := bs.FindRegistryConnections(ctx, configuration.OrganizationId, *configuration.ProjectId)
		if err != nil {
			return "", err
		}
		found := false
		for _, refreshed := range registries {
			if refreshed.Id == registry.Id && refreshed.Url == registry.Url {
				registry = refreshed
				found = true
				break
			}
		}
		if !found {
			return "", errors.New("DigitalOcean registry changed or is no longer available")
		}
		if registry.Auth.Server != "registry.digitalocean.com" || registry.Auth.Username == "" || registry.Auth.Password == "" {
			return "", errors.New("invalid DigitalOcean registry credentials")
		}
		digest, err := dockerutil.PushAuthorizedDigest(ctx, image, destination, registry.Auth.Server, registry.Auth.Username, registry.Auth.Password)
		if errors.Is(err, dockerutil.ErrRegistryAuthentication) {
			continue
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(destination, ":"+tag) + "@" + digest, nil
	}
	return "", dockerutil.ErrRegistryAuthentication
}
