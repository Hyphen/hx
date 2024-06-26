package inmemory

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type InMemoryRepo struct {
	mem      map[string]string
	filePath string
}

func New() *InMemoryRepo {
	repo := &InMemoryRepo{
		mem:      make(map[string]string),
		filePath: " env_vars.json",
	}

	if err := repo.loadFromFile(); err != nil {
		os.Exit(1)
	}

	return repo
}

func (m *InMemoryRepo) UploadEnvVariable(env, encryptedVars string) error {
	m.mem[env] = encryptedVars
	if err := m.saveToFile(); err != nil {
		return fmt.Errorf("error saving to file: %w", err)
	}
	return nil
}

func (m *InMemoryRepo) GetEncryptedVariables(env string) (string, error) {
	encryptedVars, exists := m.mem[env]
	if !exists {
		return "", fmt.Errorf("environment %s not found", env)
	}
	return encryptedVars, nil
}

func (m *InMemoryRepo) saveToFile() error {
	data, err := json.MarshalIndent(m.mem, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data to JSON: %w", err)
	}
	if err := ioutil.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing data to file: %w", err)
	}
	return nil
}

func (m *InMemoryRepo) loadFromFile() error {
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return nil // No file to load from, start fresh
	}
	data, err := ioutil.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("error reading data from file: %w", err)
	}
	if err := json.Unmarshal(data, &m.mem); err != nil {
		return fmt.Errorf("error unmarshaling data from JSON: %w", err)
	}
	return nil
}
