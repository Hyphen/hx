package code

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemToolExecutor_ReadWriteListAndSearch(t *testing.T) {
	workspaceRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte("{\n  \"name\": \"demo\"\n}\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "src"), 0o755))
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(workspaceRoot, "src", "main.ts"),
			[]byte("console.log('hello')\nconst port = 3000\n"),
			0o644,
		),
	)

	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	listResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "list_files",
			Arguments: `{"path":".","max_depth":2}`,
		},
	})
	require.NoError(t, err)
	listResult := listResultAny.(listFilesResult)
	assert.Equal(t, ".", listResult.Path)
	assert.Contains(t, listResult.Entries, workspaceFileEntry{Path: "package.json", Type: "file"})
	assert.Contains(t, listResult.Entries, workspaceFileEntry{Path: "src", Type: "directory"})
	assert.Contains(t, listResult.Entries, workspaceFileEntry{Path: "src/main.ts", Type: "file"})

	readResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "read_file",
			Arguments: `{"path":"src/main.ts","start_line":2,"end_line":2}`,
		},
	})
	require.NoError(t, err)
	readResult := readResultAny.(readFileResult)
	assert.Equal(t, "src/main.ts", readResult.Path)
	assert.Equal(t, "const port = 3000", readResult.Content)
	assert.Equal(t, 2, readResult.StartLine)
	assert.Equal(t, 2, readResult.EndLine)

	writeResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "write_file",
			Arguments: `{"path":"Dockerfile","content":"FROM node:20-alpine"}`,
		},
	})
	require.NoError(t, err)
	writeResult := writeResultAny.(writeFileResult)
	assert.Equal(t, "Dockerfile", writeResult.Path)
	assert.Equal(t, len([]byte("FROM node:20-alpine")), writeResult.BytesWritten)

	dockerfileContent, err := os.ReadFile(filepath.Join(workspaceRoot, "Dockerfile"))
	require.NoError(t, err)
	assert.Equal(t, "FROM node:20-alpine", string(dockerfileContent))

	searchResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "search",
			Arguments: `{"pattern":"port","path":"src"}`,
		},
	})
	require.NoError(t, err)
	searchResult := searchResultAny.(searchResult)
	require.Len(t, searchResult.Matches, 1)
	assert.Equal(t, "src/main.ts", searchResult.Matches[0].Path)
	assert.Equal(t, 2, searchResult.Matches[0].Line)
	assert.Equal(t, "const port = 3000", searchResult.Matches[0].Content)
}

func TestFilesystemToolExecutor_BlocksPathTraversal(t *testing.T) {
	workspaceRoot := t.TempDir()
	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	_, err = executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "read_file",
			Arguments: `{"path":"../secret.txt"}`,
		},
	})

	assert.ErrorContains(t, err, "path must stay within the workspace root")
}

func TestFilesystemToolExecutor_BlocksSymlinkEscapes(t *testing.T) {
	workspaceRoot := t.TempDir()
	outsideRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(outsideRoot, "secret.txt"), []byte("top-secret"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(outsideRoot, "linked-dir"), 0o755))

	symlinkOrSkip(t, filepath.Join(outsideRoot, "secret.txt"), filepath.Join(workspaceRoot, "secret-link.txt"))
	symlinkOrSkip(t, filepath.Join(outsideRoot, "linked-dir"), filepath.Join(workspaceRoot, "linked-dir"))

	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	_, err = executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "read_file",
			Arguments: `{"path":"secret-link.txt"}`,
		},
	})
	assert.ErrorContains(t, err, "path must stay within the workspace root")

	_, err = executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "write_file",
			Arguments: `{"path":"linked-dir/Dockerfile","content":"FROM scratch"}`,
		},
	})
	assert.ErrorContains(t, err, "path must stay within the workspace root")
}

func TestFilesystemToolExecutor_SkipsSymlinksDuringWalks(t *testing.T) {
	workspaceRoot := t.TempDir()
	outsideRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte("{\"name\":\"demo\"}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outsideRoot, "credentials.txt"), []byte("super-secret-token"), 0o644))
	symlinkOrSkip(t, filepath.Join(outsideRoot, "credentials.txt"), filepath.Join(workspaceRoot, "credentials-link.txt"))

	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	listResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "list_files",
			Arguments: `{"path":".","max_depth":2}`,
		},
	})
	require.NoError(t, err)
	listResult := listResultAny.(listFilesResult)
	assert.NotContains(t, listResult.Entries, workspaceFileEntry{Path: "credentials-link.txt", Type: "file"})

	searchResultAny, err := executor.executeToolCall(DockerfileToolCall{
		Type: "function",
		Function: DockerfileToolFunction{
			Name:      "search",
			Arguments: `{"path":".","pattern":"super-secret-token"}`,
		},
	})
	require.NoError(t, err)
	searchResult := searchResultAny.(searchResult)
	assert.Empty(t, searchResult.Matches)
}

func TestFilesystemToolExecutor_BlocksWritesToProtectedPaths(t *testing.T) {
	workspaceRoot := t.TempDir()
	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	testCases := []string{
		".git/config",
		".env",
		".env.production",
		".github/workflows/release.yml",
		".ssh/config",
		".aws/credentials",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			_, err := executor.executeToolCall(DockerfileToolCall{
				Type: "function",
				Function: DockerfileToolFunction{
					Name:      "write_file",
					Arguments: `{"path":"` + path + `","content":"test"}`,
				},
			})
			assert.ErrorContains(t, err, "writing to protected path")
		})
	}
}

func TestFilesystemToolExecutor_LogsToolArgumentsAndResultSummary(t *testing.T) {
	workspaceRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte("{\"name\":\"demo\"}\n"), 0o644))

	executor, err := NewFilesystemToolExecutor(workspaceRoot)
	require.NoError(t, err)

	var logs []string
	results := executor.ExecuteToolCalls(
		[]DockerfileToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: DockerfileToolFunction{
					Name:      "list_files",
					Arguments: `{"path":".","max_depth":1}`,
				},
			},
		},
		func(message string) {
			logs = append(logs, message)
		},
	)

	require.Len(t, results, 1)
	require.Len(t, logs, 2)
	assert.Contains(t, logs[0], "Executing call-1:list_files(path=. max_depth=1")
	assert.Contains(t, logs[1], "Tool result call-1:list_files output=path=. entry_count=")
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are not supported in this environment: %v", err)
	}
}
