package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/dockerutil"
)

const cloudflareRepositoryComponent = `[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*`

var cloudflarePushReference = regexp.MustCompile(`^registry\.cloudflare\.com/([a-f0-9]{32}/(?:` + cloudflareRepositoryComponent + `/` + cloudflareRepositoryComponent + `|p-[a-f0-9]{24}-a-[a-f0-9]{24})):([a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127})$`)
var manifestDigest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

var cloudflareRepositoryName = regexp.MustCompile(`^` + cloudflareRepositoryComponent + `$`)

func isCloudflareRegistry(registryURL string) bool {
	return strings.HasPrefix(registryURL, "registry.cloudflare.com/")
}

func cloudflareImageURI(registryURL, projectAlternateID, appAlternateID, tag string) (string, error) {
	if !cloudflareRepositoryName.MatchString(projectAlternateID) || !cloudflareRepositoryName.MatchString(appAlternateID) {
		return "", errors.New("Cloudflare uploads require valid project and app alternate IDs")
	}
	uri := registryURL + "/" + projectAlternateID + "/" + appAlternateID + ":" + tag
	if !cloudflarePushReference.MatchString(uri) {
		return "", errors.New("invalid Cloudflare image destination")
	}
	return uri, nil
}

func (bs *BuildService) pushCloudflare(ctx context.Context, configuration config.Config, registry models.ContainerRegistry, image, tag string) (string, error) {
	if configuration.ProjectId == nil || configuration.ProjectAlternateId == nil || configuration.AppAlternateId == nil {
		return "", errors.New("Cloudflare uploads require project and app alternate IDs in .hx configuration")
	}
	destination, err := cloudflareImageURI(registry.Url, *configuration.ProjectAlternateId, *configuration.AppAlternateId, tag)
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
			return "", errors.New("container registry changed or is no longer available")
		}
		if registry.Auth.Server != "registry.cloudflare.com" || registry.Auth.Username == "" || registry.Auth.Password == "" {
			return "", errors.New("invalid Cloudflare registry credentials")
		}
		err = dockerutil.PushAuthorized(ctx, image, destination, registry.Auth.Server, registry.Auth.Username, registry.Auth.Password)
		if errors.Is(err, dockerutil.ErrRegistryAuthentication) {
			continue
		}
		if err != nil {
			return "", err
		}
		client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		uri, err := resolveCloudflareDigest(ctx, client, destination, registry)
		if errors.Is(err, dockerutil.ErrRegistryAuthentication) {
			continue
		}
		return uri, err
	}
	return "", dockerutil.ErrRegistryAuthentication
}

func resolveCloudflareDigest(ctx context.Context, client interface {
	Do(*http.Request) (*http.Response, error)
}, uri string, registry models.ContainerRegistry) (string, error) {
	reference := cloudflarePushReference.FindStringSubmatch(uri)
	if len(reference) != 3 {
		return "", errors.New("invalid Cloudflare upload destination")
	}
	selector := reference[2]
	for attempt := 0; attempt < 2; attempt++ {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://registry.cloudflare.com/v2/"+reference[1]+"/manifests/"+selector, nil)
		request.SetBasicAuth(registry.Auth.Username, registry.Auth.Password)
		request.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")
		response, err := client.Do(request)
		if err != nil {
			return "", errors.New("Cloudflare manifest lookup failed")
		}
		content, readErr := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024+1))
		response.Body.Close()
		if response.StatusCode == http.StatusUnauthorized {
			return "", dockerutil.ErrRegistryAuthentication
		}
		if response.StatusCode != http.StatusOK || readErr != nil || len(content) > 4*1024*1024 {
			return "", errors.New("Cloudflare manifest is unavailable")
		}
		hash := sha256.Sum256(content)
		digest := "sha256:" + hex.EncodeToString(hash[:])
		if strings.HasPrefix(selector, "sha256:") && selector != digest {
			return "", errors.New("Cloudflare manifest digest mismatch")
		}
		var manifest struct {
			SchemaVersion int    `json:"schemaVersion"`
			MediaType     string `json:"mediaType"`
			Manifests     []struct {
				Digest   string `json:"digest"`
				Platform struct {
					OS           string `json:"os"`
					Architecture string `json:"architecture"`
				} `json:"platform"`
			} `json:"manifests"`
		}
		if json.Unmarshal(content, &manifest) != nil || manifest.SchemaVersion != 2 {
			return "", errors.New("invalid Cloudflare image manifest")
		}
		switch manifest.MediaType {
		case "application/vnd.oci.image.manifest.v1+json", "application/vnd.docker.distribution.manifest.v2+json":
			return "registry.cloudflare.com/" + reference[1] + "@" + digest, nil
		case "application/vnd.oci.image.index.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json":
			selected := ""
			for _, item := range manifest.Manifests {
				if item.Platform.OS == "linux" && item.Platform.Architecture == "amd64" {
					if selected != "" || !manifestDigest.MatchString(item.Digest) {
						return "", errors.New("ambiguous linux/amd64 image manifest")
					}
					selected = item.Digest
				}
			}
			if selected == "" {
				return "", errors.New("Cloudflare image requires a linux/amd64 manifest")
			}
			selector = selected
		default:
			return "", errors.New("unsupported Cloudflare image manifest")
		}
	}
	return "", errors.New("nested Cloudflare image indexes are unsupported")
}
