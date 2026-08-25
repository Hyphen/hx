package appcloud

import (
	"compress/gzip"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanUploads(t *testing.T) {
	t.Run("returns_posix_relpaths_sorted_and_skips_noise", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>hi</h1>"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref"), 0o644))

		files, err := planUploads(dir)

		require.NoError(t, err)
		rels := []string{}
		for _, f := range files {
			rels = append(rels, f.rel)
		}
		assert.Equal(t, []string{"assets/app.js", "index.html"}, rels)
	})

	t.Run("keeps_dotfiles_that_are_not_skipped", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".well-known"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".well-known", "security.txt"), []byte("x"), 0o644))

		files, err := planUploads(dir)

		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, ".well-known/security.txt", files[0].rel)
	})
}

func TestBuildBatch(t *testing.T) {
	t.Run("gzips_each_part_and_sets_content_encoding_and_filename", func(t *testing.T) {
		dir := t.TempDir()
		body := []byte("<!doctype html><h1>hello</h1>")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), body, 0o644))
		files, err := planUploads(dir)
		require.NoError(t, err)

		buf, contentType, raw, gz, err := buildBatch(files)

		require.NoError(t, err)
		assert.Equal(t, int64(len(body)), raw)
		assert.Greater(t, gz, int64(0))

		// Parse the multipart body back and verify the part metadata + that
		// the payload is genuinely gzip of the original file.
		_, params, err := mime.ParseMediaType(contentType)
		require.NoError(t, err)
		mr := multipart.NewReader(buf, params["boundary"])
		part, err := mr.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "index.html", part.FileName())
		assert.Equal(t, "gzip", part.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/html", firstMediaType(t, part.Header.Get("Content-Type")))

		gzr, err := gzip.NewReader(part)
		require.NoError(t, err)
		got, err := io.ReadAll(gzr)
		require.NoError(t, err)
		assert.Equal(t, body, got)
	})
}

func firstMediaType(t *testing.T, ct string) string {
	t.Helper()
	mt, _, err := mime.ParseMediaType(ct)
	require.NoError(t, err)
	return mt
}

func TestUploadDirectory(t *testing.T) {
	t.Run("uploads_all_files_in_a_single_batch", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("a"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "style.css"), []byte("b"), 0o644))
		stub := &uploadStub{}

		n, err := UploadDirectory(stub, "the_app_id", "abcd1234", dir, 100, nil)

		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, "the_app_id", stub.appID)
		assert.Equal(t, "abcd1234", stub.hex)
		assert.Equal(t, 1, stub.calls)
	})

	t.Run("splits_into_batches_by_batch_size", func(t *testing.T) {
		dir := t.TempDir()
		for _, n := range []string{"a", "b", "c"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, n+".txt"), []byte(n), 0o644))
		}
		stub := &uploadStub{}

		n, err := UploadDirectory(stub, "app", "hex", dir, 2, nil)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, 2, stub.calls) // 3 files, batch size 2 => 2 batches
	})
}

// uploadStub captures UploadBatch calls; the other methods are unused here.
type uploadStub struct {
	AppCloudServicer
	appID string
	hex   string
	calls int
}

func (u *uploadStub) UploadBatch(appID, hex string, body io.Reader, contentType string) (UploadResponse, error) {
	u.appID, u.hex = appID, hex
	u.calls++
	_, _ = io.Copy(io.Discard, body)
	return UploadResponse{}, nil
}
