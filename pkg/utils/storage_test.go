package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileSystemStorage(t *testing.T) {
	t.Parallel()

	t.Run("creates directory and saves file", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()
		storage := FileSystemStorage{BaseDir: tempDir}

		// Test saving
		err := storage.Save("test-key", []byte("test-value"))
		assert.NoError(t, err)

		// Verify file exists and has correct content
		filePath := filepath.Join(tempDir, "test-key")
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", string(content))

		// Verify file permissions
		info, err := os.Stat(filePath)
		assert.NoError(t, err)
		// File permissions on Unix systems can vary, so we just check that it's restrictive
		assert.True(t, info.Mode().Perm() <= 0o600)
	})

	t.Run("loads file content", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()
		storage := FileSystemStorage{BaseDir: tempDir}

		// Create test file
		testContent := "test-content"
		filePath := filepath.Join(tempDir, "test-key")
		err := os.WriteFile(filePath, []byte(testContent), OWNER_READ_WRITE)
		assert.NoError(t, err)

		// Test loading
		content, err := storage.Load("test-key")
		assert.NoError(t, err)
		assert.Equal(t, testContent, string(content))
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		storage := FileSystemStorage{BaseDir: tempDir}

		// Test loading non-existent file
		content, err := storage.Load("non-existent-key")
		assert.Error(t, err)
		assert.Nil(t, content)
	})

	t.Run("handles directory creation failure", func(t *testing.T) {
		// Use a path that cannot be created (e.g., under a file instead of directory)
		tempDir := t.TempDir()

		// Create a file where we want to create a directory
		filePath := filepath.Join(tempDir, "blocking-file")
		err := os.WriteFile(filePath, []byte("content"), 0644)
		assert.NoError(t, err)

		// Try to create storage with the file as base directory
		storage := FileSystemStorage{BaseDir: filePath}

		err = storage.Save("test-key", []byte("test-value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})
}
