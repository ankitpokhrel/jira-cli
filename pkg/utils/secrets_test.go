package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock storage for testing
type mockStorage struct {
	savedKey   string
	savedValue []byte
	loadReturn []byte
	loadError  error
	saveError  error
}

func (m *mockStorage) Save(key string, value []byte) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.savedKey = key
	m.savedValue = value
	return nil
}

func (m *mockStorage) Load(key string) ([]byte, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.loadReturn, nil
}

func TestSecretOperations(t *testing.T) {
	t.Parallel()

	t.Run("secret string representation", func(t *testing.T) {
		secret := Secret{Key: "test-key", Value: "test-value"}
		assert.Equal(t, "test-value", secret.String())
	})

	t.Run("secret save with empty key", func(t *testing.T) {
		secret := Secret{Key: "", Value: "test-value"}
		storage := &mockStorage{}

		err := secret.Save(storage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key cannot be empty")
	})

	t.Run("secret save success", func(t *testing.T) {
		secret := Secret{Key: "test-key", Value: "test-value"}
		storage := &mockStorage{}

		err := secret.Save(storage)
		assert.NoError(t, err)
		assert.Equal(t, "test-key", storage.savedKey)
		assert.Equal(t, []byte("test-value"), storage.savedValue)
	})

	t.Run("secret load with empty key", func(t *testing.T) {
		secret := &Secret{}
		storage := &mockStorage{}

		err := secret.Load(storage, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key cannot be empty")
	})

	t.Run("secret load success", func(t *testing.T) {
		secret := &Secret{}
		storage := &mockStorage{
			loadReturn: []byte("loaded-value"),
		}

		err := secret.Load(storage, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-key", secret.Key)
		assert.Equal(t, "loaded-value", secret.Value)
	})

	t.Run("secret load with storage error", func(t *testing.T) {
		secret := &Secret{}
		storage := &mockStorage{
			loadError: fmt.Errorf("storage error"),
		}

		err := secret.Load(storage, "test-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage error")
	})
}
