package utils

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"

	"github.com/zalando/go-keyring"
)

// KeyRingStorage implements Storage interface using the system keyring.
// The keyring library uses (service, user) as a unique key pair.
type KeyRingStorage struct {
	User string
}

var (
	ErrKeyRingValueEmpty = errors.New("value cannot be empty")
	ErrKeyRingUserEmpty  = errors.New("user cannot be empty")
)

// NewKeyRingStorage creates a new KeyRingStorage with the provided user.
func NewKeyRingStorage(user string) *KeyRingStorage {
	return &KeyRingStorage{
		User: user,
	}
}

// Save compresses the data and stores it in the system keyring.
func (ks KeyRingStorage) Save(key string, value []byte) error {
	compressedData, err := compressData(value)
	if err != nil {
		return err
	}

	if key == "" {
		return ErrKeyRingValueEmpty
	}
	if ks.User == "" {
		return ErrKeyRingUserEmpty
	}
	// Note, there is a limit to the size of the data that can be stored in the keyring. See https://github.com/zalando/go-keyring/blob/5c6f7e0ba5bf0380b4a490f2b7e41deb44b3c63e/keyring.go#L13-L16
	return keyring.Set(key, ks.User, compressedData)
}

// Load decompresses and retrieves the data from the system keyring.
func (ks KeyRingStorage) Load(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrKeyRingValueEmpty
	}
	if ks.User == "" {
		return nil, ErrKeyRingUserEmpty
	}

	secret, err := keyring.Get(key, ks.User)
	if err != nil {
		return nil, err
	}
	decompressedData, err := decompressData(secret)
	if err != nil {
		return nil, err
	}
	return decompressedData, nil
}

func compressData(value []byte) (string, error) {
	var compressed bytes.Buffer
	zlibWriter := zlib.NewWriter(&compressed)
	if _, err := zlibWriter.Write(value); err != nil {
		return "", err
	}
	if err := zlibWriter.Close(); err != nil {
		return "", err
	}

	compressedValue := compressed.String()
	return compressedValue, nil
}

func decompressData(compressedData string) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader([]byte(compressedData)))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
