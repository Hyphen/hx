package code

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDockerSessionModelViewShowsWrittenFilesMessage(t *testing.T) {
	view := GenerateDockerSessionModel{
		Done:         true,
		FilesWritten: true,
		Summary:      "Dockerfile and .dockerignore updated.",
	}.View()

	assert.Contains(t, view, "Dockerfile and .dockerignore generated! You may choose to check them in if you like.")
	assert.True(t, strings.Contains(view, "Dockerfile and .dockerignore updated."))
}

func TestGenerateDockerSessionModelViewShowsNoOpMessage(t *testing.T) {
	view := GenerateDockerSessionModel{
		Done:         true,
		FilesWritten: false,
		Summary:      "The existing Dockerfile and .dockerignore already look good.",
	}.View()

	assert.Contains(t, view, "Existing Dockerfile and .dockerignore already look good. No files were generated or changed.")
	assert.True(t, strings.Contains(view, "The existing Dockerfile and .dockerignore already look good."))
}
