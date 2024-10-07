package database

import (
	"crypto/sha256"
	"encoding/hex"

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

func (db *Database) SaveSecret(env, data string, version int) {
	secret := newSecret(data, version)
	db.Secrets[env] = secret
	Save(*db)
}

func Restore() (Database, error) {
	m, err := manifest.Restore()
	if err != nil {
		return Database{}, err
	}
	if m.Database == nil {
		newDB := Database{}
		if err := Save(newDB); err != nil {
			return Database{}, err
		}
		m.Database = newDB
	}

	db, ok := (m.Database).(Database)
	if !ok {
		return Database{}, errors.New("Unexpected type found in .hx file")
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
