package gitutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/run"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/fsutil"
)

const gitDirPath = ".git"
const gitignorePath = ".gitignore"

// Check if we're inside a git repository
func gitExists() bool {
	currentDir, err := os.Getwd()
	if err != nil {
		return false
	}

	_, found := findGitRoot(currentDir)
	return found
}

// findGitRoot searches for .git directory up the directory tree
func findGitRoot(startPath string) (string, bool) {
	currentPath := startPath
	for {
		gitPath := filepath.Join(currentPath, gitDirPath)
		if _, err := os.Stat(gitPath); err == nil {
			return currentPath, true
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			return "", false
		}
		currentPath = parent
	}
}

// Check if the .gitignore file exists, and create it if necessary
func ensureGitignoreFile() (*os.File, error) {
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		file, err := os.Create(gitignorePath)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating .gitignore")
		}
		return file, nil
	}

	file, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Error opening .gitignore")
	}
	return file, nil
}

// Check if the appendStr is already in .gitignore
func appendStrExists(file *os.File, appendStr string) (bool, error) {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == appendStr {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, errors.Wrap(err, "Error reading .gitignore")
	}
	return false, nil
}

// Check if the last character in the file is a newline
func fileEndsWithNewline(file *os.File) (bool, error) {
	stat, err := file.Stat()
	if err != nil {
		return false, errors.Wrap(err, "Error getting file info")
	}

	// Move the file pointer to the last byte
	if stat.Size() == 0 {
		return true, nil // Empty file, treat as needing a newline
	}

	buf := make([]byte, 1)
	if _, err := file.Seek(-1, io.SeekEnd); err != nil {
		return false, errors.Wrap(err, "Error seeking in file")
	}
	if _, err := file.Read(buf); err != nil {
		return false, errors.Wrap(err, "Error reading file")
	}

	return buf[0] == '\n', nil
}

func EnsureGitignore(appendStr string) error {
	if !gitExists() {
		return nil
	}

	file, err := ensureGitignoreFile()
	if err != nil {
		return err
	}
	defer file.Close()

	// Check if appendStr is already present
	found, err := appendStrExists(file, appendStr)
	if err != nil || found {
		return err
	}

	// Check if the file ends with a newline, and if not, add one
	endsWithNewline, err := fileEndsWithNewline(file)
	if err != nil {
		return err
	}
	if !endsWithNewline {
		if _, err := file.WriteString("\n"); err != nil {
			return errors.Wrap(err, "Error adding newline to .gitignore")
		}
	}

	// Now append the string
	if _, err := file.WriteString(appendStr + "\n"); err != nil {
		return errors.Wrap(err, "Error writing to .gitignore")
	}

	return nil
}

func GetLastCommitHash() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.New("unable to get current directory")
	}

	gitDir, found := findGitRoot(currentDir)

	if !found {
		return "", errors.New("not a git repository")
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gitDir
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "error getting last commit hash")
	}

	return strings.TrimSpace(string(output)), nil
}

func GetCurrentBranch() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.New("unable to get current directory")
	}
	gitDir, found := findGitRoot(currentDir)
	if !found {
		return "", errors.New("not a git repository")
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gitDir
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "error getting current branch")
	}
	return strings.TrimSpace(string(output)), nil
}

func CheckForChangesNotOnRemote() (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, errors.New("unable to get current directory")
	}

	gitDir, found := findGitRoot(currentDir)
	if !found {
		return false, nil // Not a git repository, no changes to check
	}

	// First, fetch latest remote references
	fetchCmd := exec.Command("git", "fetch")
	fetchCmd.Dir = gitDir
	if err := fetchCmd.Run(); err != nil {
		// If fetch fails, we can still check local state
		// This might happen if there's no remote configured
	}

	// Check for uncommitted changes (staged and unstaged)
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = gitDir
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return false, errors.Wrap(err, "error checking git status")
	}

	// If there are uncommitted changes, return true
	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		return true, nil
	}

	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return false, err
	}

	// Check for commits not pushed to remote
	// Try the current branch first, then fall back to main/master
	remoteBranches := []string{
		fmt.Sprintf("origin/%s", currentBranch),
		"origin/main",
		"origin/master",
	}

	var logOutput []byte
	var logErr error

	for _, remoteBranch := range remoteBranches {
		logCmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("%s..HEAD", remoteBranch))
		logCmd.Dir = gitDir
		logOutput, logErr = logCmd.Output()

		// If command succeeds, we found a valid remote branch
		if logErr == nil {
			break
		}
	}

	// If all remote branch checks failed, assume no remote exists
	if logErr != nil {
		return false, nil
	}

	// If there are commits not on remote, return true
	return len(strings.TrimSpace(string(logOutput))) > 0, nil
}

func GetRemoteUrl() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current directory: %w", err)
	}

	gitDir, found := findGitRoot(currentDir)
	if !found {
		return "", nil
	}

	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = gitDir
	output, err := cmd.Output()
	if err != nil {
		// No origin remote configured is non-fatal
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

func GetCurrentTag() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current directory: %w", err)
	}

	gitDir, found := findGitRoot(currentDir)
	if !found {
		return "", fmt.Errorf("git repository not found from %q", currentDir)
	}

	cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD")
	cmd.Dir = gitDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.ToLower(strings.TrimSpace(string(exitErr.Stderr)))
			if strings.Contains(stderr, "no tag exactly matches") || strings.Contains(stderr, "cannot describe anything") {
				return "", nil
			}
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func ParseRepoBaseUrl(remoteUrl string) (string, error) {
	remoteUrl = strings.TrimSpace(remoteUrl)

	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteUrl, "git@") {
		remoteUrl = strings.TrimPrefix(remoteUrl, "git@")
		parts := strings.SplitN(remoteUrl, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("unrecognized git remote URL format: %s", remoteUrl)
		}
		host := parts[0]
		path := strings.TrimSuffix(parts[1], ".git")
		return "https://" + host + "/" + path, nil
	}

	// HTTPS format: https://github.com/owner/repo.git
	if strings.HasPrefix(remoteUrl, "https://") || strings.HasPrefix(remoteUrl, "http://") {
		result := strings.TrimSuffix(remoteUrl, ".git")
		return result, nil
	}

	return "", fmt.Errorf("unrecognized git remote URL format: %s", remoteUrl)
}

func ApplyDiffs(diffs []run.DiffResult) {
	fs := fsutil.NewFileSystem()
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// iterate over diffs
	for _, diff := range diffs {
		// This is a delete
		if diff.To == "" {
			fullPath := filepath.Join(currentDir, diff.From)
			err := fs.Remove(fullPath)
			if err != nil {
				fmt.Printf("Error removing file %s: %v\n", fullPath, err)
			}
			continue
		}

		// This is a create or modify
		var contents []byte
		for _, chunk := range diff.Chunks {
			// TODO: handle deletes to files
			if chunk.Type != "delete" {
				contents = append(contents, []byte(chunk.Content)...)
			}
		}
		fullPath := filepath.Join(currentDir, diff.To)

		err := fs.WriteFile(fullPath, contents, 0o644)
		if err != nil {
			fmt.Printf("Error writing file %s: %v\n", fullPath, err)
		}
	}
}
