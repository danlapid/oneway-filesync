package utils

import (
	"sync"
)

// Taken from here: https://github.com/golang/go/blob/912f0750472dd4f674b69ca1616bfaf377af1805/src/sync/map_reference_test.go#L25
// RWMutexMap is an implementation of mapInterface using a sync.RWMutex.
type RWMutexMap[K comparable, V any] struct {
	mu    sync.RWMutex
	dirty map[K]V
}

func (m *RWMutexMap[K, V]) Load(key K) (value V, ok bool) {
	m.mu.RLock()
	value, ok = m.dirty[key]
	m.mu.RUnlock()
	return
}

func (m *RWMutexMap[K, V]) Store(key K, value V) {
	m.mu.Lock()
	if m.dirty == nil {
		m.dirty = make(map[K]V)
	}
	m.dirty[key] = value
	m.mu.Unlock()
}

func (m *RWMutexMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	m.mu.Lock()
	actual, loaded = m.dirty[key]
	if !loaded {
		actual = value
		if m.dirty == nil {
			m.dirty = make(map[K]V)
		}
		m.dirty[key] = value
	}
	m.mu.Unlock()
	return actual, loaded
}

func (m *RWMutexMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	m.mu.Lock()
	value, loaded = m.dirty[key]
	if !loaded {
		m.mu.Unlock()
		var v V
		return v, false
	}
	delete(m.dirty, key)
	m.mu.Unlock()
	return value, loaded
}

func (m *RWMutexMap[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.dirty, key)
	m.mu.Unlock()
}

func (m *RWMutexMap[K, V]) Range(f func(key K, value V) (shouldContinue bool)) {
	m.mu.RLock()
	keys := make([]K, 0, len(m.dirty))
	for k := range m.dirty {
		keys = append(keys, k)
	}
	m.mu.RUnlock()

	for _, k := range keys {
		v, ok := m.Load(k)
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}
