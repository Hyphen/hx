package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/errors"
)

type Database interface {
	GetSecret(key SecretKey) (Secret, bool)
	UpsertSecret(key SecretKey, data string, version int) error
}

type database struct {
	Secrets map[string]map[string]map[string]Secret `json:"secrets"`
}

// Compile-time check that *database implements the Database interface
var _ Database = (*database)(nil)

type SecretKey struct {
	ProjectId string `json:"project_id"`
	AppId     string `json:"app_id"`
	EnvName   string `json:"env_name"`
}

type Secret struct {
	Version int    `json:"version"`
	Hash    string `json:"hash"`
}

func newSecret(data string, version int) Secret {
	hash := sha256.New()

	hash.Write([]byte(data))

	hashSum := hash.Sum(nil)

	hashString := hex.EncodeToString(hashSum)
	return Secret{
		Version: version,
		Hash:    hashString,
	}
}

func (db *database) GetSecret(key SecretKey) (Secret, bool) {
	secret, ok := db.Secrets[key.ProjectId][key.AppId][key.EnvName]
	return secret, ok
}

// UpsertSecret saves a secret to the database.
// Data will be hashed before saving
func (db *database) UpsertSecret(key SecretKey, data string, version int) error {
	if db.Secrets == nil {
		db.Secrets = make(map[string]map[string]map[string]Secret)
	}

	// Initialize the nested maps if they don't exist
	if _, ok := db.Secrets[key.ProjectId]; !ok {
		db.Secrets[key.ProjectId] = make(map[string]map[string]Secret)
	}
	if _, ok := db.Secrets[key.ProjectId][key.AppId]; !ok {
		db.Secrets[key.ProjectId][key.AppId] = make(map[string]Secret)
	}

	secret := newSecret(data, version)

	db.Secrets[key.ProjectId][key.AppId][key.EnvName] = secret

	return save(*db)
}

func Restore() (Database, error) {
	m, err := config.RestoreConfig()
	if err != nil {
		return nil, err
	}

	// Check if the Database field is nil, and initialize if necessary
	if m.Database == nil {
		// Initialize a new database with an empty Secrets map
		newDB := &database{
			Secrets: make(map[string]map[string]map[string]Secret),
		}

		// Save the newly initialized database back to the manifest
		if err := save(*newDB); err != nil {
			return nil, err
		}

		// Assign the new database to the manifest
		m.Database = *newDB
	}

	// Attempt to convert m.Database to a database struct
	var db database
	switch v := m.Database.(type) {
	case database:
		db = v
	case map[string]interface{}:
		// Convert the map to JSON
		dbBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		// Unmarshal the JSON into a database struct
		if err := json.Unmarshal(dbBytes, &db); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unexpected type found in .hx file")
	}

	return &db, nil
}

func save(db database) error {
	mc, err := config.RestoreConfig()
	if err != nil {
		return err
	}
	mc.Database = db

	return config.UpsertGlobalConfig(mc)
}
