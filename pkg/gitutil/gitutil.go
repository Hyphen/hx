package gitutil

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/pkg/errors"
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
