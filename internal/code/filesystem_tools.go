package code

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Hyphen/cli/pkg/fsutil"
)

const (
	defaultListFilesMaxDepth  = 6
	defaultListFilesMaxResult = 500
	defaultSearchMaxResults   = 200
)

var errMaxResultsReached = fmt.Errorf("max tool results reached")

type FilesystemToolExecutor struct {
	root string
	fs   fsutil.FileSystem
}

type listFilesArgs struct {
	Path       string `json:"path"`
	MaxDepth   *int   `json:"max_depth"`
	MaxResults *int   `json:"max_results"`
}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine *int   `json:"start_line"`
	EndLine   *int   `json:"end_line"`
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  *bool  `json:"append"`
}

type searchArgs struct {
	Pattern       string `json:"pattern"`
	Path          string `json:"path"`
	CaseSensitive *bool  `json:"case_sensitive"`
	MaxResults    *int   `json:"max_results"`
}

type listFilesResult struct {
	Path      string               `json:"path"`
	Entries   []workspaceFileEntry `json:"entries"`
	Truncated bool                 `json:"truncated"`
}

type workspaceFileEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type readFileResult struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	TotalLines int    `json:"total_lines"`
}

type writeFileResult struct {
	Path         string `json:"path"`
	BytesWritten int    `json:"bytes_written"`
	Appended     bool   `json:"appended"`
}

type searchResult struct {
	Path      string        `json:"path"`
	Pattern   string        `json:"pattern"`
	Matches   []searchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
}

type searchMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

func NewFilesystemToolExecutor(root string) (*FilesystemToolExecutor, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	return &FilesystemToolExecutor{
		root: filepath.Clean(absoluteRoot),
		fs:   fsutil.NewFileSystem(),
	}, nil
}

func (w *FilesystemToolExecutor) ExecuteToolCalls(
	toolCalls []DockerfileToolCall,
	onVerbose func(string),
) []DockerfileToolResult {
	results := make([]DockerfileToolResult, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		if onVerbose != nil {
			onVerbose(fmt.Sprintf("Executing %s", summarizeToolCallForLog(toolCall)))
		}

		output, err := w.executeToolCall(toolCall)
		result := DockerfileToolResult{
			ToolCallID: toolCall.ID,
		}
		if err != nil {
			result.Error = err.Error()
			if onVerbose != nil {
				onVerbose(fmt.Sprintf("Tool result %s", summarizeToolResultForLog(result, &toolCall)))
			}
		} else {
			result.Output = output
			if onVerbose != nil {
				onVerbose(fmt.Sprintf("Tool result %s", summarizeToolResultForLog(result, &toolCall)))
			}
		}
		results = append(results, result)
	}

	return results
}

func (w *FilesystemToolExecutor) executeToolCall(toolCall DockerfileToolCall) (any, error) {
	if toolCall.Type != "function" {
		return nil, fmt.Errorf("unsupported tool call type: %s", toolCall.Type)
	}

	switch toolCall.Function.Name {
	case "list_files":
		return w.listFiles(toolCall.Function.Arguments)
	case "read_file":
		return w.readFile(toolCall.Function.Arguments)
	case "write_file":
		return w.writeFile(toolCall.Function.Arguments)
	case "search":
		return w.search(toolCall.Function.Arguments)
	default:
		return nil, fmt.Errorf("unsupported tool: %s", toolCall.Function.Name)
	}
}

func (w *FilesystemToolExecutor) listFiles(rawArguments string) (listFilesResult, error) {
	var args listFilesArgs
	if err := json.Unmarshal([]byte(rawArguments), &args); err != nil {
		return listFilesResult{}, fmt.Errorf("invalid list_files arguments: %w", err)
	}

	maxDepth := defaultListFilesMaxDepth
	if args.MaxDepth != nil {
		maxDepth = *args.MaxDepth
	}
	if maxDepth < 0 {
		return listFilesResult{}, fmt.Errorf("max_depth must be greater than or equal to 0")
	}

	maxResults := defaultListFilesMaxResult
	if args.MaxResults != nil {
		maxResults = *args.MaxResults
	}
	if maxResults <= 0 {
		return listFilesResult{}, fmt.Errorf("max_results must be greater than 0")
	}

	startPath, displayPath, err := w.resolvePath(args.Path, true)
	if err != nil {
		return listFilesResult{}, err
	}

	info, err := w.fs.Stat(startPath)
	if err != nil {
		return listFilesResult{}, fmt.Errorf("failed to inspect path: %w", err)
	}
	if !info.IsDir() {
		return listFilesResult{}, fmt.Errorf("path must be a directory")
	}

	result := listFilesResult{
		Path: displayPath,
	}

	err = filepath.WalkDir(startPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == startPath {
			return nil
		}

		relativeToStart, err := filepath.Rel(startPath, path)
		if err != nil {
			return err
		}

		depth := pathDepth(relativeToStart)
		if depth > maxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() && relativeToStart != "." && shouldSkipDir(entry.Name()) {
			return filepath.SkipDir
		}

		relativeToRoot, err := filepath.Rel(w.root, path)
		if err != nil {
			return err
		}

		entryType := "file"
		if entry.IsDir() {
			entryType = "directory"
		}

		result.Entries = append(result.Entries, workspaceFileEntry{
			Path: filepath.ToSlash(relativeToRoot),
			Type: entryType,
		})

		if len(result.Entries) >= maxResults {
			result.Truncated = true
			return errMaxResultsReached
		}

		return nil
	})
	if err != nil && err != errMaxResultsReached {
		return listFilesResult{}, fmt.Errorf("failed to list files: %w", err)
	}

	return result, nil
}

func (w *FilesystemToolExecutor) readFile(rawArguments string) (readFileResult, error) {
	var args readFileArgs
	if err := json.Unmarshal([]byte(rawArguments), &args); err != nil {
		return readFileResult{}, fmt.Errorf("invalid read_file arguments: %w", err)
	}

	fullPath, displayPath, err := w.resolvePath(args.Path, false)
	if err != nil {
		return readFileResult{}, err
	}

	data, err := w.fs.ReadFile(fullPath)
	if err != nil {
		return readFileResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	lines := splitLines(string(data))
	if len(lines) == 0 {
		return readFileResult{
			Path:       displayPath,
			Content:    "",
			StartLine:  0,
			EndLine:    0,
			TotalLines: 0,
		}, nil
	}

	startLine := 1
	if args.StartLine != nil {
		startLine = *args.StartLine
	}
	if startLine < 1 || startLine > len(lines) {
		return readFileResult{}, fmt.Errorf("start_line must be between 1 and %d", len(lines))
	}

	endLine := len(lines)
	if args.EndLine != nil {
		endLine = *args.EndLine
	}
	if endLine < startLine || endLine > len(lines) {
		return readFileResult{}, fmt.Errorf("end_line must be between %d and %d", startLine, len(lines))
	}

	return readFileResult{
		Path:       displayPath,
		Content:    strings.Join(lines[startLine-1:endLine], "\n"),
		StartLine:  startLine,
		EndLine:    endLine,
		TotalLines: len(lines),
	}, nil
}

func (w *FilesystemToolExecutor) writeFile(rawArguments string) (writeFileResult, error) {
	var args writeFileArgs
	if err := json.Unmarshal([]byte(rawArguments), &args); err != nil {
		return writeFileResult{}, fmt.Errorf("invalid write_file arguments: %w", err)
	}

	fullPath, displayPath, err := w.resolvePath(args.Path, false)
	if err != nil {
		return writeFileResult{}, err
	}
	if displayPath == "." {
		return writeFileResult{}, fmt.Errorf("path must point to a file")
	}

	if err := w.fs.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return writeFileResult{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	appended := args.Append != nil && *args.Append
	if appended {
		file, err := w.fs.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return writeFileResult{}, fmt.Errorf("failed to open file for append: %w", err)
		}
		defer file.Close()

		bytesWritten, err := file.WriteString(args.Content)
		if err != nil {
			return writeFileResult{}, fmt.Errorf("failed to append file: %w", err)
		}

		return writeFileResult{
			Path:         displayPath,
			BytesWritten: bytesWritten,
			Appended:     true,
		}, nil
	}

	if err := w.fs.WriteFile(fullPath, []byte(args.Content), 0o644); err != nil {
		return writeFileResult{}, fmt.Errorf("failed to write file: %w", err)
	}

	return writeFileResult{
		Path:         displayPath,
		BytesWritten: len([]byte(args.Content)),
		Appended:     false,
	}, nil
}

func (w *FilesystemToolExecutor) search(rawArguments string) (searchResult, error) {
	var args searchArgs
	if err := json.Unmarshal([]byte(rawArguments), &args); err != nil {
		return searchResult{}, fmt.Errorf("invalid search arguments: %w", err)
	}
	if strings.TrimSpace(args.Pattern) == "" {
		return searchResult{}, fmt.Errorf("pattern is required")
	}

	maxResults := defaultSearchMaxResults
	if args.MaxResults != nil {
		maxResults = *args.MaxResults
	}
	if maxResults <= 0 {
		return searchResult{}, fmt.Errorf("max_results must be greater than 0")
	}

	pattern := args.Pattern
	if args.CaseSensitive == nil || !*args.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	matcher, err := regexp.Compile(pattern)
	if err != nil {
		return searchResult{}, fmt.Errorf("invalid regex pattern: %w", err)
	}

	startPath, displayPath, err := w.resolvePath(args.Path, true)
	if err != nil {
		return searchResult{}, err
	}

	result := searchResult{
		Path:    displayPath,
		Pattern: args.Pattern,
	}

	info, err := w.fs.Stat(startPath)
	if err != nil {
		return searchResult{}, fmt.Errorf("failed to inspect search path: %w", err)
	}

	if !info.IsDir() {
		if err := w.searchFile(startPath, matcher, maxResults, &result); err != nil {
			return searchResult{}, err
		}
		return result, nil
	}

	err = filepath.WalkDir(startPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == startPath {
			return nil
		}

		if entry.IsDir() && shouldSkipDir(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}

		if err := w.searchFile(path, matcher, maxResults, &result); err != nil {
			if err == errMaxResultsReached {
				result.Truncated = true
			}
			return err
		}

		return nil
	})
	if err != nil && err != errMaxResultsReached {
		return searchResult{}, fmt.Errorf("failed to search files: %w", err)
	}

	return result, nil
}

func (w *FilesystemToolExecutor) searchFile(
	fullPath string,
	matcher *regexp.Regexp,
	maxResults int,
	result *searchResult,
) error {
	data, err := w.fs.ReadFile(fullPath)
	if err != nil {
		return err
	}
	if !utf8.Valid(data) {
		return nil
	}

	lines := splitLines(string(data))
	relativePath, err := filepath.Rel(w.root, fullPath)
	if err != nil {
		return err
	}

	for index, line := range lines {
		if !matcher.MatchString(line) {
			continue
		}

		result.Matches = append(result.Matches, searchMatch{
			Path:    filepath.ToSlash(relativePath),
			Line:    index + 1,
			Content: line,
		})
		if len(result.Matches) >= maxResults {
			return errMaxResultsReached
		}
	}

	return nil
}

func (w *FilesystemToolExecutor) resolvePath(rawPath string, allowEmpty bool) (string, string, error) {
	cleanedInput := strings.TrimSpace(rawPath)
	if cleanedInput == "" {
		if !allowEmpty {
			return "", "", fmt.Errorf("path is required")
		}
		cleanedInput = "."
	}

	cleanedInput = strings.TrimLeft(strings.ReplaceAll(cleanedInput, "\\", "/"), "/")
	if cleanedInput == "" {
		cleanedInput = "."
	}

	candidatePath := filepath.Clean(filepath.Join(w.root, filepath.FromSlash(cleanedInput)))
	relativePath, err := filepath.Rel(w.root, candidatePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path must stay within the workspace root")
	}

	resolvedRoot, err := filepath.EvalSymlinks(w.root)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	resolvedCandidatePath, err := resolvePathWithExistingAncestor(candidatePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	resolvedRelativePath, err := filepath.Rel(resolvedRoot, resolvedCandidatePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}
	if resolvedRelativePath == ".." || strings.HasPrefix(resolvedRelativePath, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path must stay within the workspace root")
	}

	displayPath := filepath.ToSlash(relativePath)
	if displayPath == "" {
		displayPath = "."
	}

	return resolvedCandidatePath, displayPath, nil
}

func resolvePathWithExistingAncestor(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	suffix := make([]string, 0)
	current := cleanPath

	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolvedCurrent, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}

			resolvedPath := resolvedCurrent
			for i := len(suffix) - 1; i >= 0; i-- {
				resolvedPath = filepath.Join(resolvedPath, suffix[i])
			}
			return filepath.Clean(resolvedPath), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}

		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}
func pathDepth(relativePath string) int {
	if relativePath == "." || relativePath == "" {
		return 0
	}
	return len(strings.Split(filepath.ToSlash(relativePath), "/"))
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".next", "dist", "build", "coverage", ".turbo":
		return true
	default:
		return false
	}
}

func splitLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	if strings.HasSuffix(normalized, "\n") {
		return lines[:len(lines)-1]
	}
	return lines
}
