package envvars

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/secretkey"
)

// EnvironmentVarsData represents environment variables data.
type EnvironmentVarsData struct {
	Size           string `json:"size"`
	CountVariables int    `json:"countVariables"`
	Data           string `json:"data"`
}

// GetEnvironmentVarsData processes the environment variables from the given file.
func New(fileName string) (EnvironmentVarsData, error) {
	var data EnvironmentVarsData

	file, err := os.Open(fileName)
	if err != nil {
		return data, err
	}
	defer file.Close()

	var contentBuilder strings.Builder
	scanner := bufio.NewScanner(file)
	countVariables := 0

	for scanner.Scan() {
		line := scanner.Text()
		contentBuilder.WriteString(line + "\n")
		if strings.Contains(line, "=") {
			countVariables++
		}
	}

	if err := scanner.Err(); err != nil {
		return data, err
	}

	content := contentBuilder.String()
	data.Size = strconv.Itoa(len(content)) + " bytes" // Convert size to string and format it
	data.CountVariables = countVariables
	data.Data = content

	return data, nil
}
func (e *EnvironmentVarsData) EncryptData(key secretkey.SecretKeyer) error {
	encryptData, err := key.Encrypt(e.Data)
	if err != nil {
		return err
	}
	e.Data = encryptData

	return nil
}

type EnvironmentInformation struct {
	Size           string `json:"size"`
	CountVariables int    `json:"countVariables"`
	Data           string `json:"data"`
	AppId          string `json:"appId"`
	EnvId          string `json:"envId"`
}
