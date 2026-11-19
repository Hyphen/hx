package build

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/dockerutil"
)

func stageStaticImage(directory string, files []uploadFile) (string, error) {
	hash := sha256.New()
	for _, file := range files {
		if file.Path == ".nfabric-not-found.html" || file.Path == "_headers" || file.Path == "_redirects" || file.Path == "_worker.js" || file.Path == "_worker.bundle" || file.Path == "_routes.json" || strings.HasPrefix(file.Path, "functions/") {
			return "", errors.New("DigitalOcean static sites do not support provider control files or server functions")
		}
		target := filepath.Join(directory, "site", filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "", err
		}
		if err := stageStaticImageFile(target, file); err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%s\x00%d\x00%s\n", file.Path, file.Size, file.SHA256)
	}
	if err := os.WriteFile(filepath.Join(directory, "Dockerfile"), []byte("FROM scratch\nCOPY site/ /site/\n"), 0600); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func stageStaticImageFile(target string, file uploadFile) error {
	compressed, err := os.Open(file.gzipPath)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		return err
	}
	defer reader.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer output.Close()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(output, hash), reader)
	if err != nil {
		return err
	}
	if size != file.Size || fmt.Sprintf("%x", hash.Sum(nil)) != file.SHA256 {
		return errors.New("staged static file changed")
	}
	return output.Close()
}

func (bs *BuildService) uploadDigitalOceanSite(ctx context.Context, orgID, appID, appName, registryID string, files []uploadFile, capability uploadCapability, opts Options) (*models.StaticArtifact, error) {
	if err := dockerutil.CheckDocker(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	directory, err := os.MkdirTemp("", "hx-static-image-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	tag, err := stageStaticImage(directory, files)
	if err != nil {
		return nil, err
	}
	registryURL := capability.Registry.Url
	destination, err := digitalOceanImageURI(registryURL, appName, tag)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.Split(strings.TrimPrefix(registryURL, "registry.digitalocean.com/"), "/")[1], "static-") {
		return nil, errors.New("DigitalOcean static image requires a static project repository")
	}
	image, _, err := dockerutil.BuildWithContext(ctx, directory, "hx-static", tag, opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("build static image: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		// Build time can outlast the upload credential, so request it immediately before pushing.
		capability, err = bs.requestSiteUpload(ctx, orgID, appID, map[string]any{"registryId": registryID, "files": files})
		if err != nil {
			return nil, err
		}
		registry := capability.Registry
		if capability.Provider != "digitalOcean" || capability.AppID != appID || registry.Id != registryID || registry.Url != registryURL || registry.Auth.Server != "registry.digitalocean.com" || registry.Auth.Username == "" || registry.Auth.Password == "" {
			return nil, errors.New("DigitalOcean static registry changed or returned invalid credentials")
		}
		digest, err := dockerutil.PushAuthorizedDigest(ctx, image, destination, registry.Auth.Server, registry.Auth.Username, registry.Auth.Password)
		if errors.Is(err, dockerutil.ErrRegistryAuthentication) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &models.StaticArtifact{Provider: "digitalOcean", RegistryID: registryID, URI: registryURL + "/" + appName + "@" + digest}, nil
	}
	return nil, dockerutil.ErrRegistryAuthentication
}
