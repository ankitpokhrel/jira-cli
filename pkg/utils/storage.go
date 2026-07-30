package utils

import (
	"encoding/json"
)

type Storage interface {
	Save(key string, value []byte) error
	Load(key string) ([]byte, error)
}

// SaveJSON saves a typed value as JSON using the provided storage.
func SaveJSON[T any](storage Storage, key string, value T) error {
	data, err := json.Marshal(value)
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
