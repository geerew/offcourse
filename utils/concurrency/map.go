package concurrency

import "sync"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Map is a thread-safe concurrent map with generic type support
type Map[K comparable, V any] struct {
	data map[K]V
	lock sync.RWMutex
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewMap creates a new concurrent map
func NewMap[K comparable, V any]() Map[K, V] {
	return Map[K, V]{
		data: make(map[K]V),
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Has checks if a key exists in the map
func (m *Map[K, V]) Has(key K) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.data[key]
	return ok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Get gets a value from the map
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ret, ok := m.data[key]
	return ret, ok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetOrSet sets a value, creating it if it does not yet exist
//
// It returns the value and a boolean indicating if it was created
func (m *Map[K, V]) GetOrSet(key K, val V) (V, bool) {
	return m.GetOrSetFn(key, func() V { return val })
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetOrSetFn sets a value, creating it via a set function if it does not yet exist
//
// It returns the value and a boolean indicating if it was created
func (m *Map[K, V]) GetOrSetFn(key K, setFn func() V) (V, bool) {
	m.lock.RLock()

	if ret, ok := m.data[key]; ok {
		m.lock.RUnlock()
		return ret, false
	}
	m.lock.RUnlock()

	m.lock.Lock()
	defer m.lock.Unlock()

	if ret, ok := m.data[key]; ok {
		return ret, false
	}

	val := setFn()
	m.data[key] = val

	return val, true
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Set sets a value in the map
//
// If the key already exists, this will overwrite the value
func (m *Map[K, V]) Set(key K, val V) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = val
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Remove deletes a value from the map
func (m *Map[K, V]) Remove(key K) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.data, key)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAndRemove gets and deletes a value from the map
//
// It returns the value and a boolean indicating if it was found
func (m *Map[K, V]) GetAndRemove(key K) (V, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	val, ok := m.data[key]
	delete(m.data, key)

	return val, ok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Len returns the number of elements in the map
func (m *Map[K, V]) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.data)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Keys returns a slice of all keys in the map
func (m *Map[K, V]) Keys() []K {
	m.lock.RLock()
	defer m.lock.RUnlock()

	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}

	return keys
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Values returns a slice of all values in the map
func (m *Map[K, V]) Values() []V {
	m.lock.RLock()
	defer m.lock.RUnlock()

	values := make([]V, 0, len(m.data))
	for _, v := range m.data {
		values = append(values, v)
	}

	return values
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Clear removes all elements from the map
func (m *Map[K, V]) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data = make(map[K]V)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ForEach is a helper function that calls a function against each key/value pair in the map
//
// It holds the write lock for the duration of the function call
func (m *Map[K, V]) ForEach(fn func(K, V)) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for k, v := range m.data {
		fn(k, v)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Range is a helper function that calls a function against each key/value pair in the map. It stops
// iterating when the function returns false
//
// It holds the read lock for the duration of the function call
func (m *Map[K, V]) Range(fn func(K, V) bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for k, v := range m.data {
		if !fn(k, v) {
			break
		}
	}
}
