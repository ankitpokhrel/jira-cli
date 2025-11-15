package utils

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

// KeyRingStorage implements Storage interface using the system keyring.
// The keyring library uses (service, user) as a unique key pair.
type KeyRingStorage struct {
	User string
}

// NewKeyRingStorage creates a new KeyRingStorage with the provided user.
func NewKeyRingStorage(user string) *KeyRingStorage {
	return &KeyRingStorage{
		User: user,
	}
}

// Save stores the value in the system keyring.
// The key parameter is used as the keyring's service field.
func (ks KeyRingStorage) Save(key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if ks.User == "" {
		return fmt.Errorf("user cannot be empty")
	}

	return keyring.Set(key, ks.User, string(value))
}

// Load retrieves the value from the system keyring.
func (ks KeyRingStorage) Load(key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}
	if ks.User == "" {
		return nil, fmt.Errorf("user cannot be empty")
	}

	secret, err := keyring.Get(key, ks.User)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}
