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
		if strings.Contains(line, "=") {
			contentBuilder.WriteString(line + "\n")
			countVariables++
		}
	}

	if err := scanner.Err(); err != nil {
		return data, err
	}

	content := contentBuilder.String()
	data.Size = strconv.Itoa(len(content)) + " bytes"
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

func (e *EnvironmentVarsData) EnvVarsToArray() []string {
	return strings.Split(e.Data, "\n")
}

type EnvironmentInformation struct {
	Size           string `json:"size"`
	CountVariables int    `json:"countVariables"`
	Data           string `json:"data"`
	AppId          string `json:"appId"`
	EnvId          string `json:"envId"`
}

func (e *EnvironmentInformation) ToEnvironmentVarsData() EnvironmentVarsData {
	return EnvironmentVarsData{
		Size:           e.Size,
		CountVariables: e.CountVariables,
		Data:           e.Data,
	}
}
