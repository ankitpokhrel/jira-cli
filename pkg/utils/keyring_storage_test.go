package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

const (
	testUser = "test-user"
)

func TestNewKeyRingStorage(t *testing.T) {
	storage := NewKeyRingStorage("test-user")
	assert.NotNil(t, storage)
	assert.Equal(t, "test-user", storage.User)
}

func TestKeyRingStorage(t *testing.T) {
	// Use mock keyring for all tests
	keyring.MockInit()

	t.Run("saves and loads value", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "test-key"
		testValue := []byte("test-value")

		// Test saving
		err := storage.Save(testKey, testValue)
		assert.NoError(t, err)

		// Test loading
		loadedValue, err := storage.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, testValue, loadedValue)
	})

	t.Run("handles empty key on save", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}

		err := storage.Save("", []byte("test-value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})

	t.Run("handles empty key on load", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}

		_, err := storage.Load("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})

	t.Run("handles empty user on save", func(t *testing.T) {
		storage := &KeyRingStorage{User: ""}

		err := storage.Save("test-key", []byte("test-value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user cannot be empty")
	})

	t.Run("handles empty user on load", func(t *testing.T) {
		storage := &KeyRingStorage{User: ""}

		_, err := storage.Load("test-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user cannot be empty")
	})

	t.Run("handles non-existent key", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}

		// Try to load a key that doesn't exist
		_, err := storage.Load("non-existent-key")
		assert.Error(t, err)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "overwrite-test-key"

		// Save initial value
		err := storage.Save(testKey, []byte("initial-value"))
		assert.NoError(t, err)

		// Overwrite with new value
		err = storage.Save(testKey, []byte("new-value"))
		assert.NoError(t, err)

		// Verify new value is loaded
		loadedValue, err := storage.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, []byte("new-value"), loadedValue)
	})

	t.Run("handles binary data", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "binary-key"
		// Test with binary data including null bytes
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}

		err := storage.Save(testKey, binaryData)
		assert.NoError(t, err)

		loadedValue, err := storage.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, binaryData, loadedValue)
	})

	t.Run("handles large values", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "large-value-key"
		// Create a large value (1KB)
		largeValue := make([]byte, 1024)
		for i := range largeValue {
			largeValue[i] = byte(i % 256)
		}

		err := storage.Save(testKey, largeValue)
		assert.NoError(t, err)

		loadedValue, err := storage.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, largeValue, loadedValue)
	})

	t.Run("handles multiple keys in same service", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}

		// Save multiple keys
		keys := []string{"key1", "key2", "key3"}
		values := [][]byte{[]byte("value1"), []byte("value2"), []byte("value3")}

		for i, key := range keys {
			err := storage.Save(key, values[i])
			assert.NoError(t, err)
		}

		// Verify all keys can be loaded independently
		for i, key := range keys {
			loadedValue, err := storage.Load(key)
			assert.NoError(t, err)
			assert.Equal(t, values[i], loadedValue)
		}
	})

	t.Run("handles special characters in keys", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		specialKeys := []string{
			"key-with-dashes",
			"key.with.dots",
			"key_with_underscores",
			"key@with@symbols",
		}

		for _, key := range specialKeys {
			value := []byte(fmt.Sprintf("value-for-%s", key))
			err := storage.Save(key, value)
			assert.NoError(t, err)

			loadedValue, err := storage.Load(key)
			assert.NoError(t, err)
			assert.Equal(t, value, loadedValue)
		}
	})

	t.Run("different keys are isolated", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		key1 := "isolation-key-1"
		key2 := "isolation-key-2"

		// Save different values with different keys
		err := storage.Save(key1, []byte("value1"))
		assert.NoError(t, err)

		err = storage.Save(key2, []byte("value2"))
		assert.NoError(t, err)

		// Verify each key returns its own value
		value1, err := storage.Load(key1)
		assert.NoError(t, err)
		assert.Equal(t, []byte("value1"), value1)

		value2, err := storage.Load(key2)
		assert.NoError(t, err)
		assert.Equal(t, []byte("value2"), value2)
	})

	t.Run("different users are isolated", func(t *testing.T) {
		storage1 := &KeyRingStorage{User: "user1"}
		storage2 := &KeyRingStorage{User: "user2"}
		testKey := "isolation-key"

		// Save different values for different users with the same key
		err := storage1.Save(testKey, []byte("value1"))
		assert.NoError(t, err)

		err = storage2.Save(testKey, []byte("value2"))
		assert.NoError(t, err)

		// Verify each user returns its own value
		value1, err := storage1.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, []byte("value1"), value1)

		value2, err := storage2.Load(testKey)
		assert.NoError(t, err)
		assert.Equal(t, []byte("value2"), value2)
	})
}

func TestKeyRingStorageImplementsInterface(t *testing.T) {
	// This test ensures KeyRingStorage implements the Storage interface
	var _ Storage = &KeyRingStorage{}
	var _ Storage = KeyRingStorage{}
}

// TestKeyRingStorageWithHelpers tests the SaveJSON and LoadJSON helper functions
func TestKeyRingStorageWithHelpers(t *testing.T) {
	keyring.MockInit()

	t.Run("SaveJSON and LoadJSON with struct", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "json-test-key"

		type TestData struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		original := TestData{
			Name:  "test",
			Value: 42,
		}

		// Test SaveJSON
		err := SaveJSON(storage, testKey, original)
		assert.NoError(t, err)

		// Test LoadJSON
		loaded, err := LoadJSON[TestData](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, original, loaded)
	})

	t.Run("SaveJSON and LoadJSON with map", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}
		testKey := "json-map-key"

		original := map[string]interface{}{
			"key1": "value1",
			"key2": float64(123),
			"key3": true,
		}

		err := SaveJSON(storage, testKey, original)
		assert.NoError(t, err)

		loaded, err := LoadJSON[map[string]interface{}](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, original, loaded)
	})

	t.Run("LoadJSON handles non-existent key", func(t *testing.T) {
		storage := &KeyRingStorage{User: testUser}

		type TestData struct {
			Name string `json:"name"`
		}

		_, err := LoadJSON[TestData](storage, "non-existent-json-key")
		assert.Error(t, err)
	})
}
