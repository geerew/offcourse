package concurrency

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap_BasicOps(t *testing.T) {
	m := NewMap[string, int]()

	// Test successfully setting and getting a key/value pair
	m.Set("a", 1)
	v, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, v)

	// Test setting a value for an existing key
	v, created := m.GetOrSet("a", 2)
	require.False(t, created)
	require.Equal(t, 1, v)

	// Test setting a value for a new key
	v, created = m.GetOrSet("b", 3)
	require.True(t, created)
	require.Equal(t, 3, v)

	// Test setting a value for an existing key via a set function
	v, created = m.GetOrSetFn("b", func() int { return 2 + 2 })
	require.False(t, created)
	require.Equal(t, 3, v)

	// Test setting a value for a new key via a set function
	v, created = m.GetOrSetFn("c", func() int { return 2 + 3 })
	require.True(t, created)
	require.Equal(t, 5, v)

	// Test getting the length, keys, and values
	require.Equal(t, 3, m.Len())
	keys := m.Keys()
	require.ElementsMatch(t, []string{"a", "b", "c"}, keys)
	vals := m.Values()
	require.ElementsMatch(t, []int{1, 3, 5}, vals)

	// Test getting and removing a key/value pair
	v, ok = m.GetAndRemove("b")
	require.True(t, ok)
	require.Equal(t, 3, v)
	require.False(t, m.Has("b"))

	// Test removing a key/value pair
	m.Remove("c")
	require.False(t, m.Has("c"))

	// Test clearing the map
	m.Clear()
	require.Equal(t, 0, m.Len())

	// Test running a function against each key/value pair
	m2 := NewMap[string, *int]()
	a := 1
	b := 2
	c := 3
	m2.Set("a", &a)
	m2.Set("b", &b)
	m2.Set("c", &c)
	m2.ForEach(func(k string, v *int) {
		*v += 3
	})
	require.Equal(t, 4, a)
	require.Equal(t, 5, b)
	require.Equal(t, 6, c)

	// Test Range stops when the callback returns false
	m3 := NewMap[string, *int]()
	a = 1
	b = 2
	c = 3
	m3.Set("a", &a)
	m3.Set("b", &b)
	m3.Set("c", &c)
	n := 0
	m3.Range(func(k string, v *int) bool {
		n++
		*v += 3
		return n < 2
	})
	require.Equal(t, 12, a+b+c)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMap_GetOrCreate_ConcurrentSingleCreation(t *testing.T) {
	m := NewMap[string, int]()
	var createdCount int
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			_, created := m.GetOrSetFn("x", func() int { createdCount++; return 42 })
			_ = created
		}()
	}
	wg.Wait()
	v, ok := m.Get("x")
	require.True(t, ok)
	require.Equal(t, 42, v)
	require.Equal(t, 1, createdCount)
}
