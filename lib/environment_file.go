package vaulted

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/miquella/xdg"
)

type EnvironmentFile struct {
	Method     string  `json:"method"`
	Details    Details `json:"details,omitempty"`
	Ciphertext []byte  `json:"ciphertext"`
}

func readEnvironmentFile(name string) (*EnvironmentFile, error) {
	existing := xdg.CACHE_HOME.Find(filepath.Join("vaulted", name))
	if existing == "" {
		return nil, os.ErrNotExist
	}

	f, err := os.Open(existing)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	ef := EnvironmentFile{}
	err = d.Decode(&ef)
	if err != nil {
		return nil, err
	}

	return &ef, nil
}

func writeEnvironmentFile(name string, environmentFile *EnvironmentFile) error {
	pathname := xdg.CACHE_HOME.Join("vaulted")
	err := os.MkdirAll(pathname, 0700)
	if err != nil {
		return err
	}

	filename := xdg.CACHE_HOME.Join(filepath.Join("vaulted", name))
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(environmentFile)
	if err != nil {
		return err
	}

	return nil
}

func removeEnvironment(name string) error {
	existing := xdg.CACHE_HOME.Find(filepath.Join("vaulted", name))
	if existing == "" {
		return os.ErrNotExist
	}

	return os.Remove(existing)
}
