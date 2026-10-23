package build

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
)

func ValidateInput(buildType, directory, dockerfile string) error {
	switch buildType {
	case "docker":
		if directory != "" {
			return fmt.Errorf("a site directory is only supported with --type static")
		}
	case "static":
		if dockerfile != "" {
			return fmt.Errorf("--dockerfile cannot be used with --type static")
		}
		if directory == "" {
			return fmt.Errorf("--type static requires the site directory as the final argument")
		}
		info, err := os.Lstat(filepath.Clean(directory))
		if err != nil {
			return fmt.Errorf("cannot read site directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("static build path must be a directory, not a file or symlink")
		}
	default:
		return fmt.Errorf("invalid --type %q: expected static or docker", buildType)
	}
	return nil
}

type SiteRegistry struct {
	ID                      string                                   `json:"id"`
	Type                    string                                   `json:"type"`
	Status                  string                                   `json:"status"`
	Organization            models.OrganizationReference             `json:"organization"`
	Project                 models.ProjectReference                  `json:"project"`
	Entity                  models.ConnectionEntity                  `json:"entity"`
	OrganizationIntegration models.ConnectionOrganizationIntegration `json:"organizationIntegration"`
}

func (bs *BuildService) FindSiteRegistries(organizationID, projectID string) ([]*SiteRegistry, error) {
	query := url.Values{"projectIds": {projectID}, "types": {"SiteRegistry"}, "statuses": {"Ready"}, "pageSize": {"100"}}
	var registries []*SiteRegistry
	for page := 1; ; page++ {
		query.Set("pageNum", fmt.Sprint(page))
		endpoint := fmt.Sprintf("%s/api/organizations/%s/integrations/connections?%s", bs.baseUrl, url.PathEscape(organizationID), query.Encode())
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := bs.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		var result struct {
			Data []SiteRegistry `json:"data"`
		}
		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return errors.HandleHTTPError(resp)
			}
			return json.NewDecoder(resp.Body).Decode(&result)
		}()
		if err != nil {
			return nil, fmt.Errorf("find site registry: %w", err)
		}
		for _, registry := range result.Data {
			if registry.Type != "SiteRegistry" || registry.Status != "Ready" || registry.Organization.ID != organizationID || registry.Project.ID != projectID || registry.Entity.Type != "Project" || registry.Entity.Id != projectID || registry.OrganizationIntegration.Id == "" || registry.ID == "" {
				continue
			}
			if registry.OrganizationIntegration.Type != "cloudflare" {
				return nil, fmt.Errorf("unsupported site registry provider %q for registry %s", registry.OrganizationIntegration.Type, registry.ID)
			}
			registries = append(registries, &registry)
		}
		if len(result.Data) < 100 {
			break
		}
	}
	if len(registries) == 0 {
		return nil, fmt.Errorf("no ready site registries found for project %s; check project integrations", projectID)
	}
	return registries, nil
}

type uploadFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	gzipPath string
}

type uploadCapability struct {
	Provider   string     `json:"provider"`
	AppID      string     `json:"appId"`
	Uploads    []r2Upload `json:"uploads"`
	Endpoint   string     `json:"endpoint"`
	BucketName string     `json:"bucketName"`
	Manifest   struct {
		Key     string `json:"key"`
		Content string `json:"content"`
	} `json:"manifest"`
	Credentials r2Credentials `json:"credentials"`
	Revision    string        `json:"revision"`
	URI         string        `json:"uri"`
	ExpiresAt   time.Time     `json:"expiresAt"`
	Limits      struct {
		MaxFiles     int   `json:"maxFiles"`
		MaxBytes     int64 `json:"maxBytes"`
		MaxFileBytes int64 `json:"maxFileBytes"`
	} `json:"limits"`
}

func prepareSite(directory, stagingDir string) ([]uploadFile, error) {
	var files []uploadFile
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == stagingDir {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !utf8.ValidString(relative) || len(relative) > 1024 || strings.ContainsAny(relative, "\\%?#") || strings.IndexFunc(relative, unicode.IsControl) >= 0 {
			return fmt.Errorf("unsupported site path %q", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("site path %q must be a regular file; symlinks are not supported", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		openedInfo, err := input.Stat()
		if err != nil {
			return err
		}
		if !os.SameFile(info, openedInfo) {
			return fmt.Errorf("site file %q changed while opening it", relative)
		}
		compressed, err := os.CreateTemp(stagingDir, "file-*.gz")
		if err != nil {
			return err
		}
		defer compressed.Close()
		hash := sha256.New()
		writer := gzip.NewWriter(compressed)
		size, err := io.Copy(io.MultiWriter(writer, hash), input)
		closeErr := writer.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err := compressed.Close(); err != nil {
			return err
		}
		files = append(files, uploadFile{Path: relative, Size: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil)), gzipPath: compressed.Name()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("site directory contains no files")
	}
	return files, nil
}

func (bs *BuildService) uploadStaticSites(ctx context.Context, printer *cprint.CPrinter, orgID, projectID, appID, appName string, opts Options) ([]*models.StaticArtifact, error) {
	registries, err := bs.FindSiteRegistries(orgID, projectID)
	if err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp("", "hx-site-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			printer.Warning("Could not remove temporary site files: " + err.Error())
		}
	}()
	printer.Print("Preparing static site files")
	files, err := prepareSite(opts.Directory, staging)
	if err != nil {
		return nil, fmt.Errorf("prepare static site: %w", err)
	}
	var sites []*models.StaticArtifact
	for _, registry := range registries {
		site, err := bs.uploadStaticSite(ctx, orgID, appID, appName, registry, files, opts)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, nil
}

func (bs *BuildService) uploadStaticSite(ctx context.Context, orgID, appID, appName string, registry *SiteRegistry, files []uploadFile, opts Options) (*models.StaticArtifact, error) {
	payload := struct {
		RegistryID    string       `json:"registryId"`
		Files         []uploadFile `json:"files"`
		EnvironmentID string       `json:"environmentId,omitempty"`
		PreviewHash   string       `json:"previewHash,omitempty"`
	}{registry.ID, files, opts.EnvironmentID, opts.PreviewHash}
	capability, err := bs.requestSiteUpload(ctx, orgID, appID, payload)
	if err != nil {
		return nil, err
	}
	return bs.uploadR2Site(ctx, orgID, appID, appName, registry.ID, files, capability)
}

func (bs *BuildService) requestSiteUpload(ctx context.Context, orgID, appID string, payload any) (uploadCapability, error) {
	var capability uploadCapability
	data, err := json.Marshal(payload)
	if err != nil {
		return capability, err
	}
	endpoint := fmt.Sprintf("%s/api/organizations/%s/apps/%s/builds/uploads", bs.baseUrl, url.PathEscape(orgID), url.PathEscape(appID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return capability, err
	}
	resp, err := bs.httpClient.Do(req)
	if err != nil {
		return capability, fmt.Errorf("site upload authorization unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return capability, errors.HandleHTTPError(resp)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 12*1024*1024)).Decode(&capability); err != nil {
		return capability, fmt.Errorf("invalid site upload authorization response")
	}
	return capability, nil
}
