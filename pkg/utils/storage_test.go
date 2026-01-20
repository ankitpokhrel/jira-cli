package utils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testStorage is a mock storage for testing SaveJSON and LoadJSON.
type testStorage struct {
	data      map[string][]byte
	saveError error
	loadError error
}

func newTestStorage() *testStorage {
	return &testStorage{
		data: make(map[string][]byte),
	}
}

func (m *testStorage) Save(key string, value []byte) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.data[key] = value
	return nil
}

func (m *testStorage) Load(key string) ([]byte, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	value, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func TestSaveJSON(t *testing.T) {
	t.Run("saves simple struct as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-struct"

		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		person := Person{Name: "John Doe", Age: 30}

		err := SaveJSON(storage, testKey, person)
		assert.NoError(t, err)

		// Verify the data was saved correctly
		savedData, ok := storage.data[testKey]
		assert.True(t, ok)

		var loaded Person
		err = json.Unmarshal(savedData, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, person, loaded)
	})

	t.Run("saves nested struct as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-nested"

		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type User struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		user := User{
			Name: "Jane Doe",
			Address: Address{
				Street: "123 Main St",
				City:   "Springfield",
			},
		}

		err := SaveJSON(storage, testKey, user)
		assert.NoError(t, err)

		savedData := storage.data[testKey]
		var loaded User
		err = json.Unmarshal(savedData, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, user, loaded)
	})

	t.Run("saves map as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-map"

		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
			"key4": []string{"a", "b", "c"},
		}

		err := SaveJSON(storage, testKey, data)
		assert.NoError(t, err)

		savedData := storage.data[testKey]
		var loaded map[string]interface{}
		err = json.Unmarshal(savedData, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, "value1", loaded["key1"])
		assert.Equal(t, float64(123), loaded["key2"]) // JSON numbers are float64
		assert.Equal(t, true, loaded["key3"])
	})

	t.Run("saves slice as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-slice"

		data := []string{"apple", "banana", "cherry"}

		err := SaveJSON(storage, testKey, data)
		assert.NoError(t, err)

		savedData := storage.data[testKey]
		var loaded []string
		err = json.Unmarshal(savedData, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, data, loaded)
	})

	t.Run("saves pointer to struct as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-pointer"

		type Config struct {
			Enabled bool   `json:"enabled"`
			Timeout int    `json:"timeout"`
			Name    string `json:"name"`
		}

		config := &Config{
			Enabled: true,
			Timeout: 30,
			Name:    "test-config",
		}

		err := SaveJSON(storage, testKey, config)
		assert.NoError(t, err)

		savedData := storage.data[testKey]
		var loaded Config
		err = json.Unmarshal(savedData, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, *config, loaded)
	})

	t.Run("saves empty struct as JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-empty"

		type Empty struct{}
		empty := Empty{}

		err := SaveJSON(storage, testKey, empty)
		assert.NoError(t, err)
		assert.Equal(t, []byte("{}"), storage.data[testKey])
	})

	t.Run("saves nil pointer as null", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-nil"

		var nilPointer *string

		err := SaveJSON(storage, testKey, nilPointer)
		assert.NoError(t, err)
		assert.Equal(t, []byte("null"), storage.data[testKey])
	})

	t.Run("handles storage save error", func(t *testing.T) {
		storage := &testStorage{
			saveError: fmt.Errorf("storage save failed"),
		}
		testKey := "test-key"

		type Simple struct {
			Value string `json:"value"`
		}

		err := SaveJSON(storage, testKey, Simple{Value: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage save failed")
	})

	t.Run("handles unmarshalable types", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-unmarshalable"

		// Channels cannot be marshaled to JSON
		invalidData := make(chan int)

		err := SaveJSON(storage, testKey, invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json")
	})
}

func TestLoadJSON(t *testing.T) {
	t.Run("loads simple struct from JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-struct"

		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		expected := Person{Name: "John Doe", Age: 30}
		jsonData, _ := json.Marshal(expected)
		storage.data[testKey] = jsonData

		loaded, err := LoadJSON[Person](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, expected, loaded)
	})

	t.Run("loads nested struct from JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-nested"

		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type User struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		expected := User{
			Name: "Jane Doe",
			Address: Address{
				Street: "123 Main St",
				City:   "Springfield",
			},
		}
		jsonData, _ := json.Marshal(expected)
		storage.data[testKey] = jsonData

		loaded, err := LoadJSON[User](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, expected, loaded)
	})

	t.Run("loads map from JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-map"

		expected := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		jsonData, _ := json.Marshal(expected)
		storage.data[testKey] = jsonData

		loaded, err := LoadJSON[map[string]string](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, expected, loaded)
	})

	t.Run("loads slice from JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-slice"

		expected := []int{1, 2, 3, 4, 5}
		jsonData, _ := json.Marshal(expected)
		storage.data[testKey] = jsonData

		loaded, err := LoadJSON[[]int](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, expected, loaded)
	})

	t.Run("loads pointer to struct from JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-pointer"

		type Config struct {
			Enabled bool `json:"enabled"`
			Timeout int  `json:"timeout"`
		}

		expected := Config{Enabled: true, Timeout: 30}
		jsonData, _ := json.Marshal(expected)
		storage.data[testKey] = jsonData

		loaded, err := LoadJSON[*Config](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, expected, *loaded)
	})

	t.Run("returns zero value on non-existent key", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "non-existent"

		type Simple struct {
			Value string `json:"value"`
		}

		loaded, err := LoadJSON[Simple](storage, testKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found")
		assert.Equal(t, Simple{}, loaded) // Should return zero value
	})

	t.Run("handles storage load error", func(t *testing.T) {
		storage := &testStorage{
			loadError: fmt.Errorf("storage load failed"),
		}
		testKey := "test-key"

		type Simple struct {
			Value string `json:"value"`
		}

		loaded, err := LoadJSON[Simple](storage, testKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage load failed")
		assert.Equal(t, Simple{}, loaded)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-invalid"

		storage.data[testKey] = []byte("invalid json{]")

		type Simple struct {
			Value string `json:"value"`
		}

		loaded, err := LoadJSON[Simple](storage, testKey)
		assert.Error(t, err)
		assert.Equal(t, Simple{}, loaded)
	})

	t.Run("handles JSON with missing fields", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-partial"

		type Complete struct {
			Required string `json:"required"`
			Optional string `json:"optional"`
		}

		// JSON with only one field
		storage.data[testKey] = []byte(`{"required":"value"}`)

		loaded, err := LoadJSON[Complete](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, "value", loaded.Required)
		assert.Equal(t, "", loaded.Optional) // Should have zero value
	})

	t.Run("handles JSON with extra fields", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-extra"

		type Simple struct {
			Field string `json:"field"`
		}

		// JSON with extra fields that aren't in the struct
		storage.data[testKey] = []byte(`{"field":"value","extra":"ignored"}`)

		loaded, err := LoadJSON[Simple](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, "value", loaded.Field)
	})
}

func TestSaveAndLoadJSON_Integration(t *testing.T) {
	t.Run("round-trip with complex struct", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-roundtrip"

		type Tag struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}

		type Article struct {
			Title     string                 `json:"title"`
			Content   string                 `json:"content"`
			Author    string                 `json:"author"`
			Tags      []Tag                  `json:"tags"`
			Published bool                   `json:"published"`
			Views     int                    `json:"views"`
			Metadata  map[string]interface{} `json:"metadata"`
		}

		original := Article{
			Title:   "Test Article",
			Content: "This is a test article with some content.",
			Author:  "John Doe",
			Tags: []Tag{
				{Name: "go", Color: "blue"},
				{Name: "testing", Color: "green"},
			},
			Published: true,
			Views:     1234,
			Metadata: map[string]interface{}{
				"category": "technology",
				"priority": float64(5),
			},
		}

		// Save
		err := SaveJSON(storage, testKey, original)
		assert.NoError(t, err)

		// Load
		loaded, err := LoadJSON[Article](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, original.Title, loaded.Title)
		assert.Equal(t, original.Content, loaded.Content)
		assert.Equal(t, original.Author, loaded.Author)
		assert.Equal(t, len(original.Tags), len(loaded.Tags))
		assert.Equal(t, original.Published, loaded.Published)
		assert.Equal(t, original.Views, loaded.Views)
	})

	t.Run("round-trip with different storage implementations", func(t *testing.T) {
		testKey := "test-storage"

		type Data struct {
			Value string `json:"value"`
			Count int    `json:"count"`
		}

		original := Data{Value: "test", Count: 42}

		// Test with mock storage
		mockStore := newTestStorage()
		err := SaveJSON(mockStore, testKey, original)
		assert.NoError(t, err)

		loaded, err := LoadJSON[Data](mockStore, testKey)
		assert.NoError(t, err)
		assert.Equal(t, original, loaded)

		// Test with filesystem storage
		tempDir := t.TempDir()
		fsStore := FileSystemStorage{BaseDir: tempDir}
		err = SaveJSON(&fsStore, testKey, original)
		assert.NoError(t, err)

		loaded, err = LoadJSON[Data](&fsStore, testKey)
		assert.NoError(t, err)
		assert.Equal(t, original, loaded)
	})

	t.Run("overwrite existing JSON data", func(t *testing.T) {
		storage := newTestStorage()
		testKey := "test-overwrite"

		type Version struct {
			Number int    `json:"number"`
			Name   string `json:"name"`
		}

		v1 := Version{Number: 1, Name: "first"}
		v2 := Version{Number: 2, Name: "second"}

		// Save v1
		err := SaveJSON(storage, testKey, v1)
		assert.NoError(t, err)

		loaded, err := LoadJSON[Version](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, v1, loaded)

		// Overwrite with v2
		err = SaveJSON(storage, testKey, v2)
		assert.NoError(t, err)

		loaded, err = LoadJSON[Version](storage, testKey)
		assert.NoError(t, err)
		assert.Equal(t, v2, loaded)
	})
}
