package config

import (
	"encoding/json"
	"testing"

	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticConfigFS struct {
	fsutil.FileSystem
	local, global string
}

func (f staticConfigFS) ReadFile(path string) ([]byte, error) {
	if path == ManifestConfigFile {
		return []byte(f.local), nil
	}
	return []byte(f.global), nil
}

func TestStaticBuildConfig(t *testing.T) {
	for _, test := range []struct{ name, local, global, expected string }{
		{"omitted", `{}`, `{}`, ""},
		{"configured", `{"static":{"headers":[{"path":"/*","set":{"X-Test":"value"}}],"spa":{"enabled":true}}}`, `{}`, `{"headers":[{"path":"/*","set":{"X-Test":"value"}}],"spa":{"enabled":true}}`},
		{"explicit clearing", `{"static":{"headers":[],"redirects":[],"spa":{"enabled":false},"notFoundPage":null}}`, `{"static":{"spa":{"enabled":true},"notFoundPage":"404.html"}}`, `{"headers":[],"redirects":[],"spa":{"enabled":false},"notFoundPage":null}`},
		{"empty local options", `{"static":{}}`, `{"static":{"headers":[{"path":"/*","set":{"X-Test":"global"}}]}}`, `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := FS
			FS = staticConfigFS{FileSystem: original, local: test.local, global: test.global}
			t.Cleanup(func() { FS = original })
			cfg, err := RestoreConfigFromFile(ManifestConfigFile)
			require.NoError(t, err)
			encoded, err := json.Marshal(cfg)
			require.NoError(t, err)
			var saved map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &saved))
			if test.expected == "" {
				assert.Nil(t, cfg.Static)
				assert.NotContains(t, saved, "static")
			} else {
				assert.JSONEq(t, test.expected, string(cfg.Static))
				assert.JSONEq(t, test.expected, string(saved["static"]))
			}
		})
	}
}
