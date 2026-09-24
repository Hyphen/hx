package appcloud

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"

	"github.com/Hyphen/cli/pkg/errors"
)

// skipNames are directory/file names never uploaded (VCS + build noise +
// macOS metadata). Other dotfiles (`.well-known/...`, `.htaccess`) are kept —
// they're real deploy artifacts.
var skipNames = map[string]bool{
	".git":         true,
	"node_modules": true,
	".DS_Store":    true,
}

type plannedFile struct {
	abs string
	rel string // POSIX (/-separated) relative path — the object-store key
}

// planUploads enumerates `dir` into a stable, sorted upload list.
func planUploads(dir string) ([]plannedFile, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve %s", dir)
	}
	var files []plannedFile
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if skipNames[info.Name()] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		relOS, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrapf(err, "relativize %s", path)
		}
		rel := filepath.ToSlash(relOS)
		files = append(files, plannedFile{abs: path, rel: rel})
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walk %s", root)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, nil
}

// buildBatch gzips each file and assembles one multipart/form-data body. Each
// part carries the filename (the relative path), a guessed Content-Type, and
// Content-Encoding: gzip — the signal management uses to know the part is
// gzipped. Returns the body, its content type, and raw/gzipped byte totals.
func buildBatch(files []plannedFile) (*bytes.Buffer, string, int64, int64, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	var raw, gz int64
	for _, f := range files {
		data, err := os.ReadFile(f.abs)
		if err != nil {
			return nil, "", 0, 0, errors.Wrapf(err, "read %s", f.abs)
		}
		raw += int64(len(data))
		gzipped, err := gzipBytes(data)
		if err != nil {
			return nil, "", 0, 0, errors.Wrapf(err, "gzip %s", f.rel)
		}
		gz += int64(len(gzipped))

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, f.rel))
		h.Set("Content-Type", contentType(f.rel))
		h.Set("Content-Encoding", "gzip")
		part, err := w.CreatePart(h)
		if err != nil {
			return nil, "", 0, 0, errors.Wrapf(err, "multipart part for %s", f.rel)
		}
		if _, err := part.Write(gzipped); err != nil {
			return nil, "", 0, 0, errors.Wrapf(err, "write part for %s", f.rel)
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", 0, 0, errors.Wrap(err, "finalize multipart body")
	}
	return &buf, w.FormDataContentType(), raw, gz, nil
}

// UploadDirectory walks `dir`, gzips every file, and uploads them to `hex` in
// batches of `batchSize`. Returns the number of files uploaded.
func UploadDirectory(svc AppCloudServicer, appID, hex, dir string, batchSize int, progress func(msg string)) (int, error) {
	if batchSize < 1 {
		return 0, errors.New("batch size must be >= 1")
	}
	files, err := planUploads(dir)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}
	total := len(files)
	batches := (total + batchSize - 1) / batchSize
	done := 0
	var rawTotal, gzTotal int64
	for b := 0; b < total; b += batchSize {
		end := b + batchSize
		if end > total {
			end = total
		}
		chunk := files[b:end]
		body, ct, raw, gz, err := buildBatch(chunk)
		if err != nil {
			return done, err
		}
		if _, err := svc.UploadBatch(appID, hex, body, ct); err != nil {
			// Return the server error as-is: hx's error type shows only the
			// outermost message, and HandleHTTPError's is the informative one.
			return done, err
		}
		done += len(chunk)
		rawTotal += raw
		gzTotal += gz
		if progress != nil {
			progress(fmt.Sprintf("  batch %d/%d: %d files, %s → %s gzipped",
				(b/batchSize)+1, batches, len(chunk), humanBytes(raw), humanBytes(gz)))
		}
	}
	if progress != nil {
		progress(fmt.Sprintf("uploaded %d files (%s → %s gzipped) across %d batch(es)",
			done, humanBytes(rawTotal), humanBytes(gzTotal), batches))
	}
	return done, nil
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func contentType(rel string) string {
	if ct := mime.TypeByExtension(filepath.Ext(rel)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
