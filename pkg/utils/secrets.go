package utils

import "fmt"

// Secret represents a secret value with storage capabilities
type Secret struct {
	Key   string
	Value string
}

func (s Secret) String() string {
	return s.Value
}

func (s Secret) Save(storage Storage) error {
	if s.Key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}
	return storage.Save(s.Key, []byte(s.Value))
}

func (s *Secret) Load(storage Storage, key string) error {
	if key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}

	data, err := storage.Load(key)
	if err != nil {
		return err
	}

	s.Key = key
	s.Value = string(data)
	return nil
}
