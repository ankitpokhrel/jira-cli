package filter

// Filter groups filterable config.
type Filter interface {
	Key() Key
	Val() interface{}
}

// Key represents filter key for issue.
type Key string

// Collection is a group of unique filters.
type Collection []Filter

// Get returns filter value as it is passed.
func (flt Collection) Get(key Key) interface{} {
	for _, f := range flt {
		if f.Key() == key {
			return f.Val()
		}
	}
	return nil
}

// GetInt returns filter value as an integer.
func (flt Collection) GetInt(key Key) int {
	for _, f := range flt {
		if f.Key() != key {
			continue
		}
		if v, ok := f.Val().(uint); ok {
			return int(v)
		}
	}
	return 0
}
