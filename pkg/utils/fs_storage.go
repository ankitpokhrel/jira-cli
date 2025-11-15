package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
	if !strings.HasSuffix(key, ".json") {
		key += ".json"
	}

	filePath := filepath.Join(fs.BaseDir, key)
	return os.WriteFile(filePath, value, OWNER_READ_WRITE)
}

func (fs FileSystemStorage) Load(key string) ([]byte, error) {
	filePath := filepath.Join(fs.BaseDir, key)
	return os.ReadFile(filePath)
}
