package initialize

import (
	"bufio"
	"os"
	"strings"

	"github.com/Hyphen/cli/pkg/errors"
)

func ensureGitignore(manifestSecretsFileName string) error {
	const gitDirPath = ".git"
	const gitignorePath = ".gitignore"

	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		return nil
	}

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		file, err := os.Create(gitignorePath)
		if err != nil {
			return errors.Wrap(err, "Error creating .gitignore")
		}
		defer file.Close()

		_, err = file.WriteString(manifestSecretsFileName + "\n")
		if err != nil {
			return errors.Wrap(err, "Error writing to .gitignore")
		}
		return nil
	}

	file, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "Error opening .gitignore")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == manifestSecretsFileName {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "Error reading .gitignore")
	}

	_, err = file.WriteString(manifestSecretsFileName + "\n")
	if err != nil {
		return errors.Wrap(err, "Error writing to .gitignore")
	}

	return nil
}
