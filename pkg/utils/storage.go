package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Storage interface {
	Save(key string, value []byte) error
	Load(key string) ([]byte, error)
}

const (
	OWNER_ONLY       = 0o700
	OWNER_READ_WRITE = 0o600
)

// FileSystemStorage implements Storage interface for filesystem operations.
type FileSystemStorage struct {
	// BaseDir is the directory where the storage will be saved
	BaseDir string
}

func (fs FileSystemStorage) Save(key string, value []byte) error {
	if err := os.MkdirAll(fs.BaseDir, OWNER_ONLY); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(fs.BaseDir, key)
	return os.WriteFile(filePath, value, OWNER_READ_WRITE)
}

func (fs FileSystemStorage) Load(key string) ([]byte, error) {
	filePath := filepath.Join(fs.BaseDir, key)
	return os.ReadFile(filePath)
}

// SaveJSON saves a typed value as JSON using the provided storage.
func SaveJSON[T any](storage Storage, key string, value T) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return storage.Save(key, data)
}

// LoadJSON loads a typed value from JSON using the provided storage.
func LoadJSON[T any](storage Storage, key string) (T, error) {
	var result T
	data, err := storage.Load(key)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}
