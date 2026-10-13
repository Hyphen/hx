package build

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Hyphen/cli/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/smithy-go/encoding/httpbinding"
	"golang.org/x/text/unicode/norm"
)

type r2Upload struct {
	Path        string `json:"path"`
	Key         string `json:"key"`
	ContentType string `json:"contentType"`
}

type r2Credentials struct {
	AccessKeyID     string    `json:"accessKeyId"`
	SecretAccessKey string    `json:"secretAccessKey"`
	SessionToken    string    `json:"sessionToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

var r2UploadHost = regexp.MustCompile(`^[a-f0-9]{32}\.(?:(?:eu|us|fedramp)\.)?r2\.cloudflarestorage\.com$`)
var r2BucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
var r2ObjectKey = regexp.MustCompile(`^revisions/[A-Za-z0-9_-]+/revision_[a-f0-9]{24}/(?:files/[^\\?#%\x00-\x1f\x7f]+|manifest\.json)$`)
var cloudflareRevision = regexp.MustCompile(`^[a-f0-9]{24}$`)
var siteChecksum = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validR2ObjectKey(key string) bool {
	return len(key) <= 1024 && r2ObjectKey.MatchString(key) && path.Clean(key) == key
}

func putR2Object(ctx context.Context, client *http.Client, capability uploadCapability, key, contentType string, body io.Reader, size int64, hash string) (int, error) {
	target, err := url.Parse(capability.Endpoint)
	if err != nil || target.Scheme != "https" || target.User != nil || target.Port() != "" || !r2UploadHost.MatchString(target.Hostname()) || target.Fragment != "" || target.RawQuery != "" || target.Path != "" || !r2BucketName.MatchString(capability.BucketName) || !validR2ObjectKey(key) {
		return 0, fmt.Errorf("invalid R2 upload destination")
	}
	credentials := capability.Credentials
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" || credentials.SessionToken == "" || !credentials.ExpiresAt.After(time.Now()) || credentials.ExpiresAt.After(capability.ExpiresAt) {
		return 0, fmt.Errorf("invalid R2 temporary credentials")
	}
	target.Path = "/" + capability.BucketName + "/" + key
	target.RawPath = httpbinding.EscapePath(target.Path, false)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), body)
	if err != nil {
		return 0, fmt.Errorf("invalid R2 upload request")
	}
	req.ContentLength = size
	if size == 0 {
		req.Body = http.NoBody
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Content-Sha256", hash)
	err = v4.NewSigner().SignHTTP(ctx, aws.Credentials{AccessKeyID: credentials.AccessKeyID, SecretAccessKey: credentials.SecretAccessKey, SessionToken: credentials.SessionToken}, req, hash, "s3", "auto", time.Now(), func(options *v4.SignerOptions) { options.DisableURIPathEscaping = true })
	if err != nil {
		return 0, fmt.Errorf("could not sign R2 upload")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("R2 upload request unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("R2 upload failed (HTTP %d)", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func putR2File(ctx context.Context, client *http.Client, capability uploadCapability, instruction r2Upload, file uploadFile) (int, error) {
	compressed, err := os.Open(file.gzipPath)
	if err != nil {
		return 0, fmt.Errorf("cannot open staged site file")
	}
	defer compressed.Close()
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		return 0, fmt.Errorf("cannot read staged site file")
	}
	defer reader.Close()
	return putR2Object(ctx, client, capability, instruction.Key, instruction.ContentType, io.LimitReader(reader, file.Size), file.Size, file.SHA256)
}

func (capability uploadCapability) validateR2(appID, appName, registryID string, files []uploadFile) (map[string]uploadFile, error) {
	if capability.Provider != "cloudflare" || capability.AppID != appID || !cloudflareRevision.MatchString(capability.Revision) || !capability.ExpiresAt.After(time.Now()) {
		return nil, fmt.Errorf("invalid or expired Cloudflare revision upload")
	}
	prefix := "revisions/" + appName + "/revision_" + capability.Revision + "/"
	if capability.URI != "r2://"+capability.BucketName+"/"+strings.TrimSuffix(prefix, "/") || capability.Manifest.Key != prefix+"manifest.json" || !validR2ObjectKey(capability.Manifest.Key) || len(capability.Manifest.Content) == 0 || len(capability.Manifest.Content) > 4*1024*1024 {
		return nil, fmt.Errorf("invalid R2 revision manifest")
	}
	var manifest struct {
		FormatVersion int    `json:"formatVersion"`
		Revision      string `json:"revision"`
		SHA256        string `json:"sha256"`
		AppID         string `json:"appId"`
		AppName       string `json:"appName"`
		RegistryID    string `json:"registryId"`
		Files         []struct {
			uploadFile
			ContentType string `json:"contentType"`
		} `json:"files"`
	}
	if json.Unmarshal([]byte(capability.Manifest.Content), &manifest) != nil || manifest.FormatVersion != 4 || manifest.Revision != capability.Revision || !siteChecksum.MatchString(manifest.SHA256) || manifest.AppID != appID || manifest.AppName != appName || manifest.RegistryID != registryID || len(manifest.Files) != len(files) || len(capability.Uploads) != len(files) {
		return nil, fmt.Errorf("R2 revision manifest does not match the requested build")
	}
	byPath := make(map[string]uploadFile, len(files))
	var total int64
	for _, file := range files {
		key := norm.NFC.String(file.Path)
		if _, exists := byPath[key]; exists {
			return nil, fmt.Errorf("duplicate normalized site path")
		}
		byPath[key] = file
		total += file.Size
		if file.Size > capability.Limits.MaxFileBytes {
			return nil, fmt.Errorf("site file %q exceeds upload limit of %d bytes", file.Path, capability.Limits.MaxFileBytes)
		}
	}
	if len(files) > capability.Limits.MaxFiles {
		return nil, fmt.Errorf("site exceeds registry file count limit (%d files)", capability.Limits.MaxFiles)
	}
	if total > capability.Limits.MaxBytes {
		return nil, fmt.Errorf("site exceeds registry total byte limit (%d bytes)", capability.Limits.MaxBytes)
	}
	seen := make(map[string]bool, len(files))
	for i, instruction := range capability.Uploads {
		file, exists := byPath[instruction.Path]
		entry := manifest.Files[i]
		if !exists || seen[instruction.Path] || instruction.Key != prefix+"files/"+instruction.Path || !validR2ObjectKey(instruction.Key) || entry.Path != instruction.Path || entry.Size != file.Size || entry.SHA256 != file.SHA256 || entry.ContentType != instruction.ContentType {
			return nil, fmt.Errorf("R2 upload object is outside this revision or differs from the build")
		}
		seen[instruction.Path] = true
	}
	return byPath, nil
}

func (bs *BuildService) uploadR2Site(ctx context.Context, orgID, appID, appName, registryID string, files []uploadFile, capability uploadCapability) (*models.StaticArtifact, error) {
	byPath, err := capability.validateR2(appID, appName, registryID, files)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	original := capability
	renew := func() error {
		next, err := bs.requestSiteUpload(ctx, orgID, appID, map[string]any{"registryId": registryID, "revision": original.Revision, "files": files})
		if err != nil {
			return err
		}
		if _, err = next.validateR2(appID, appName, registryID, files); err != nil {
			return err
		}
		if next.Endpoint != original.Endpoint || next.BucketName != original.BucketName || next.Revision != original.Revision || next.URI != original.URI || next.Manifest != original.Manifest {
			return fmt.Errorf("renewed revision upload scope changed")
		}
		capability = next
		return nil
	}
	put := func(upload func() (int, error)) error {
		for attempt := 0; attempt < 3; attempt++ {
			if !capability.Credentials.ExpiresAt.After(time.Now().Add(5 * time.Second)) {
				if err := renew(); err != nil {
					return err
				}
			}
			status, err := upload()
			if status != http.StatusForbidden {
				return err
			}
			if err := renew(); err != nil {
				return err
			}
		}
		return fmt.Errorf("R2 temporary credentials could not authorize the upload")
	}
	for _, instruction := range original.Uploads {
		err := put(func() (int, error) { return putR2File(ctx, client, capability, instruction, byPath[instruction.Path]) })
		if err != nil {
			return nil, err
		}
	}
	contents := []byte(capability.Manifest.Content)
	hash := fmt.Sprintf("%x", sha256.Sum256(contents))
	err = put(func() (int, error) {
		return putR2Object(ctx, client, capability, capability.Manifest.Key, "application/json", bytes.NewReader(contents), int64(len(contents)), hash)
	})
	if err != nil {
		return nil, err
	}
	return &models.StaticArtifact{Provider: "cloudflare", RegistryID: registryID, URI: capability.URI}, nil
}
