package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
)

type Database struct {
	Secrets map[string]Secret `json:"secrets"`
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

func (db *Database) GetSecret(env string) (Secret, bool) {
	secret, ok := db.Secrets[env]
	return secret, ok
}

// SaveSecret saves a secret to the Database.
// Data will be hashed before saving
func (db *Database) SaveSecret(env, data string, version int) error {
	if db.Secrets == nil {
		db.Secrets = make(map[string]Secret)
	}

	secret := newSecret(data, version)

	db.Secrets[env] = secret

	return Save(*db)
}

func Restore() (Database, error) {
	m, err := manifest.Restore()
	if err != nil {
		return Database{}, err
	}

	// Check if the Database field is nil, and initialize if necessary
	if m.Database == nil {
		// Initialize a new Database with an empty Secrets map
		newDB := Database{
			Secrets: make(map[string]Secret),
		}

		// Save the newly initialized Database back to the manifest
		if err := Save(newDB); err != nil {
			return Database{}, err
		}

		// Assign the new Database to the manifest
		m.Database = newDB
	}

	// Attempt to convert m.Database to a Database struct
	var db Database
	switch v := m.Database.(type) {
	case Database:
		db = v
	case map[string]interface{}:
		// Convert the map to JSON
		dbBytes, err := json.Marshal(v)
		if err != nil {
			return Database{}, err
		}

		// Unmarshal the JSON into a Database struct
		if err := json.Unmarshal(dbBytes, &db); err != nil {
			return Database{}, err
		}
	default:
		return Database{}, errors.New("unexpected type found in .hx file")
	}

	return db, nil
}

func Save(db Database) error {
	m, err := manifest.Restore()
	if err != nil {
		return err
	}
	m.Database = db

	return manifest.UpsertGlobalManifest(m)

}
