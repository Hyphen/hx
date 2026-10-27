package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDecodeErrorsIdentifyTheFileWithoutExposingValues(t *testing.T) {
	for _, test := range []struct {
		name, content, expected, secret string
	}{
		{"syntax", `{"hyphen_api_key":"secret-api-key",}`, "Invalid JSON", "secret-api-key"},
		{"field type", `{"hyphen_api_key":"secret-api-key","is_monorepo":"yes"}`, `Invalid value for "is_monorepo"`, "secret-api-key"},
		{"numeric overflow", `{"expiry_time":123456789012345678901234567890}`, `Invalid value for "expiry_time"`, "123456789012345678901234567890"},
		{"root type", `["secret-api-key"]`, "expected config.Config", "secret-api-key"},
	} {
		for _, location := range []string{"local", "global"} {
			t.Run(test.name+"/"+location, func(t *testing.T) {
				original := FS
				fs := staticConfigFS{local: `{}`, global: `{}`}
				filename := ManifestConfigFile
				if location == "local" {
					fs.local = test.content
				} else {
					fs.global = test.content
					filename = filepath.Join(GetGlobalDirectory(), ManifestConfigFile)
				}
				FS = fs
				t.Cleanup(func() { FS = original })

				_, err := RestoreConfig()

				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expected)
				assert.Contains(t, err.Error(), fmt.Sprintf("%q", filename))
				assert.Contains(t, err.Error(), "at byte ")
				assert.NotContains(t, err.Error(), test.secret)
				encoded, marshalErr := json.Marshal(err)
				require.NoError(t, marshalErr)
				assert.NotContains(t, string(encoded), test.secret)
			})
		}
	}
}
