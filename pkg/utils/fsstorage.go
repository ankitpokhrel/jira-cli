package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ankitpokhrel/jira-cli/pkg/terminal"
)

const (
	OWNER_ONLY       = 0o700
	OWNER_READ_WRITE = 0o600
)

// FileSystemStorage implements Storage interface for filesystem operations.
// Ideally now that we have keyring storage, we primarily use that and only fallback to filesystem storage if the keyring is not available.
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
	terminal.Warn("\nSaved credentials to owner-restricted filesystem storage")

	return os.WriteFile(filePath, value, OWNER_READ_WRITE)
}

func (fs FileSystemStorage) Load(key string) ([]byte, error) {
	if !strings.HasSuffix(key, ".json") {
		key += ".json"
	}
	filePath := filepath.Join(fs.BaseDir, key)
	terminal.Warn("\nLoaded credentials from owner-restricted filesystem storage")
	return os.ReadFile(filePath)
}
