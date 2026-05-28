package types

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully marshalling a JsonMap to JSON
func TestJsonMap_MarshalJSON(t *testing.T) {
	scenarios := []struct {
		json     JsonMap
		expected string
	}{
		{nil, "{}"},
		{JsonMap{}, `{}`},
		{JsonMap{"test1": 123, "test2": "lorem"}, `{"test1":123,"test2":"lorem"}`},
		{JsonMap{"test": []int{1, 2, 3}}, `{"test":[1,2,3]}`},
	}

	for i, s := range scenarios {
		result, err := s.json.MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, s.expected, string(result), "(%d) Expected %s, got %s", i, s.expected, string(result))
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the value of a JsonMap
func TestJsonMap_Value(t *testing.T) {
	scenarios := []struct {
		json     JsonMap
		expected driver.Value
	}{
		{nil, `{}`},
		{JsonMap{}, `{}`},
		{JsonMap{"test1": 123, "test2": "lorem"}, `{"test1":123,"test2":"lorem"}`},
		{JsonMap{"test": []int{1, 2, 3}}, `{"test":[1,2,3]}`},
	}

	for i, s := range scenarios {
		result, err := s.json.Value()
		require.NoError(t, err)
		require.Equal(t, s.expected, result, "(%d) Expected %s, got %s", i, s.expected, result)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestJsonMap_Scan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			value    any
			expected string
		}{
			{``, `{}`},
			{nil, `{}`},
			{[]byte{}, `{}`},
			{`{}`, `{}`},
			{`{"test": 1}`, `{"test":1}`},
			{[]byte(`{"test": 1}`), `{"test":1}`},
		}

		for i, s := range scenarios {
			jsonMap := JsonMap{}

			err := jsonMap.Scan(s.value)
			require.NoError(t, err, "(%d) Expected no error, got %v", i, err)
			require.Equal(t, s.expected, jsonMap.String(), "(%d) Expected %s, got %s", i, s.expected, jsonMap.String())

		}
	})

	t.Run("error", func(t *testing.T) {
		scenarios := []any{
			123,
			`""`,
			`invalid_json`,
			`"test"`,
			`1,2,3`,
			`{"test": 1`,
		}

		for i, s := range scenarios {
			jsonMap := JsonMap{}
			err := jsonMap.Scan(s)
			require.Error(t, err, "(%d) Expected error, got %v", i, err)
		}
	})
}
