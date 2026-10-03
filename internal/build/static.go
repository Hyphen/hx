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
	"mime/multipart"
	"net/http"
	"net/textproto"
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

func (bs *BuildService) FindSiteRegistry(organizationID, projectID string) (*SiteRegistry, error) {
	query := url.Values{"projectIds": {projectID}, "integrationTypes": {"hyphenCloud"}, "types": {"SiteRegistry"}, "pageSize": {"100"}}
	var found *SiteRegistry
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
			if registry.Type != "SiteRegistry" || registry.Status != "Ready" || registry.Organization.ID != organizationID || registry.Project.ID != projectID || registry.Entity.Type != "Project" || registry.Entity.Id != projectID || registry.OrganizationIntegration.Type != "hyphenCloud" || registry.OrganizationIntegration.Id == "" || registry.ID == "" {
				continue
			}
			if found != nil {
				return nil, fmt.Errorf("multiple ready HyphenCloud site registries found for project %s", projectID)
			}
			found = &registry
		}
		if len(result.Data) < 100 {
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no ready HyphenCloud site registry found for project %s; check project integrations", projectID)
	}
	return found, nil
}

type uploadFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	gzipPath string
}

type uploadCapability struct {
	SessionID   string    `json:"sessionId"`
	Revision    string    `json:"revision"`
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expiresAt"`
	UploadURL   string    `json:"uploadUrl"`
	FinalizeURL string    `json:"finalizeUrl"`
	AbortURL    string    `json:"abortUrl"`
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

func (bs *BuildService) uploadStaticSite(ctx context.Context, printer *cprint.CPrinter, orgID, projectID, appID string, opts Options) (*models.StaticArtifact, error) {
	registry, err := bs.FindSiteRegistry(orgID, projectID)
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
	payload := struct {
		RegistryID    string       `json:"registryId"`
		Files         []uploadFile `json:"files"`
		EnvironmentID string       `json:"environmentId,omitempty"`
		PreviewHash   string       `json:"previewHash,omitempty"`
	}{registry.ID, files, opts.EnvironmentID, opts.PreviewHash}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/organizations/%s/apps/%s/builds/uploads", bs.baseUrl, url.PathEscape(orgID), url.PathEscape(appID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := bs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	var capability uploadCapability
	err = func() error {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errors.HandleHTTPError(resp)
		}
		return json.NewDecoder(resp.Body).Decode(&capability)
	}()
	if err != nil {
		return nil, fmt.Errorf("request site upload: %w", err)
	}
	if err := capability.validate(files); err != nil {
		return nil, err
	}
	// Capability requests must not use HyphenHTTPClient: it injects APIX auth
	// and replaces multipart Content-Type. Redirects must not forward the token.
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	ctx, cancel := context.WithDeadline(ctx, capability.ExpiresAt)
	defer cancel()
	sealed := false
	defer func() {
		if sealed {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := capability.request(cleanupCtx, client, http.MethodDelete, capability.AbortURL, nil, "", 0, nil); err != nil {
			printer.Warning("Could not abort unfinished site upload: " + err.Error())
		}
	}()
	for _, file := range files {
		printer.PrintVerbose("Uploading " + file.Path)
		if err := capability.upload(ctx, client, file); err != nil {
			return nil, fmt.Errorf("upload %s: %w", file.Path, err)
		}
	}
	printer.Print("Finalizing static site revision")
	var receipt struct {
		OrganizationID string `json:"organizationId"`
		ProjectID      string `json:"projectId"`
		AppID          string `json:"appId"`
		Revision       string `json:"revision"`
		State          string `json:"state"`
	}
	if err := capability.request(ctx, client, http.MethodPost, capability.FinalizeURL, nil, "", 0, &receipt); err != nil {
		return nil, fmt.Errorf("finalize site upload: %w", err)
	}
	sealed = true
	if receipt.State != "sealed" || receipt.OrganizationID != orgID || receipt.ProjectID != projectID || receipt.AppID != appID || receipt.Revision != capability.Revision {
		return nil, fmt.Errorf("finalized site receipt does not match the requested app and revision; build was not registered")
	}
	return &models.StaticArtifact{RegistryID: registry.ID, Revision: receipt.Revision}, nil
}

func (capability uploadCapability) validate(files []uploadFile) error {
	if capability.Token == "" || capability.SessionID == "" || capability.Revision == "" || !capability.ExpiresAt.After(time.Now()) {
		return fmt.Errorf("missing or expired site upload capability; retry the build")
	}
	var origin string
	for _, endpoint := range []string{capability.UploadURL, capability.FinalizeURL, capability.AbortURL} {
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid site upload capability URL")
		}
		if origin != "" && origin != u.Scheme+"://"+u.Host {
			return fmt.Errorf("site upload capability URLs must share an origin")
		}
		origin = u.Scheme + "://" + u.Host
	}
	if len(files) > capability.Limits.MaxFiles {
		return fmt.Errorf("site exceeds registry file count limit (%d)", capability.Limits.MaxFiles)
	}
	var size int64
	for _, file := range files {
		if file.Size > capability.Limits.MaxFileBytes {
			return fmt.Errorf("site file %q exceeds registry file size limit", file.Path)
		}
		size += file.Size
	}
	if capability.Limits.MaxBytes <= 0 || size > capability.Limits.MaxBytes {
		return fmt.Errorf("site exceeds registry total byte limit")
	}
	return nil
}

func (capability uploadCapability) upload(ctx context.Context, client *http.Client, file uploadFile) error {
	compressed, err := os.Open(file.gzipPath)
	if err != nil {
		return err
	}
	defer compressed.Close()
	info, err := compressed.Stat()
	if err != nil {
		return err
	}
	var framing bytes.Buffer
	writer := multipart.NewWriter(&framing)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+strings.ReplaceAll(file.Path, `"`, `\"`)+`"`)
	header.Set("Content-Encoding", "gzip")
	header.Set("Content-Type", "application/octet-stream")
	if _, err := writer.CreatePart(header); err != nil {
		return err
	}
	prefix := append([]byte(nil), framing.Bytes()...)
	framing.Reset()
	if err := writer.Close(); err != nil {
		return err
	}
	length := int64(len(prefix)+framing.Len()) + info.Size()
	if length > capability.Limits.MaxBytes {
		return fmt.Errorf("compressed multipart file exceeds registry request byte limit")
	}
	body := io.MultiReader(bytes.NewReader(prefix), compressed, &framing)
	return capability.request(ctx, client, http.MethodPost, capability.UploadURL, body, writer.FormDataContentType(), length, nil)
}

func (capability uploadCapability) request(ctx context.Context, client *http.Client, method, endpoint string, body io.Reader, contentType string, length int64, result any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("invalid upload request")
	}
	req.Header.Set("Authorization", "Bearer "+capability.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.ContentLength = length
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("site upload request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.ReplaceAll(string(data), capability.Token, "[redacted]")
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("site upload token rejected or expired; retry the build (HTTP 401)")
		}
		return fmt.Errorf("site upload returned HTTP %d: %s", resp.StatusCode, message)
	}
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
