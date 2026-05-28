package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully creating a new DateTime
func Test_NowDateTime(t *testing.T) {
	now := time.Now().UTC().Format(DefaultDateLayout)
	dateTime := NowDateTime()

	require.Equal(t, dateTime.String(), now)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully parsing a various values into a DateTime
func Test_ParseDateTime(t *testing.T) {
	nowTime := time.Now().UTC()
	nowDateTime, _ := ParseDateTime(nowTime)
	nowStr := nowTime.Format(DefaultDateLayout)

	scenarios := []struct {
		value    any
		expected string
	}{
		{nil, ""},
		{"", ""},
		{"invalid", ""},
		{nowDateTime, nowStr},
		{nowTime, nowStr},
		{1641024040, "2022-01-01 08:00:40.000Z"},
		{int32(1641024040), "2022-01-01 08:00:40.000Z"},
		{int64(1641024040), "2022-01-01 08:00:40.000Z"},
		{uint(1641024040), "2022-01-01 08:00:40.000Z"},
		{uint64(1641024040), "2022-01-01 08:00:40.000Z"},
		{uint32(1641024040), "2022-01-01 08:00:40.000Z"},
		{"2022-01-01 11:23:45.678", "2022-01-01 11:23:45.678Z"},
	}

	for i, s := range scenarios {
		dt, err := ParseDateTime(s.value)

		require.Nil(t, err, "(%d) Failed to parse %v: %v", i, s.value, err)
		require.Equal(t, s.expected, dt.String(), "(%d) Expected %q, got %q", i, s.expected, dt.String())
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if DateTime is zero
func TestDateTime_IsZero(t *testing.T) {
	dateTime := DateTime{}
	require.True(t, dateTime.IsZero())

	dateTime = NowDateTime()
	require.False(t, dateTime.IsZero())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if two DateTime are the same
func TestDateTime_Equal(t *testing.T) {
	scenarios := []struct {
		dateTime1 DateTime
		dateTime2 DateTime
		expected  bool
	}{
		{DateTime{}, DateTime{}, true},
		{
			dateTime1: DateTime(time.Date(2022, 1, 1, 11, 23, 45, 678000000, time.UTC)),
			dateTime2: DateTime(time.Date(2022, 1, 1, 11, 23, 45, 678000000, time.UTC)),
			expected:  true,
		},
		{
			dateTime1: DateTime(time.Date(2022, 1, 1, 11, 23, 45, 0, time.UTC)),
			dateTime2: DateTime(time.Date(2022, 1, 1, 11, 23, 46, 0, time.UTC)),
			expected:  false,
		},
	}

	for i, s := range scenarios {
		require.Equal(t, s.expected, s.dateTime1.Equal(s.dateTime2), "(%d) Expected %v.Equal(%v) to be %v", i, s.dateTime1, s.dateTime2, s.expected)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the string representation of a DateTime
func TestDateTime_String(t *testing.T) {
	dateTime := DateTime{}
	require.Empty(t, dateTime.String())

	expected := "2022-01-01 11:23:45.678Z"
	dateTime, _ = ParseDateTime(expected)
	require.Equal(t, expected, dateTime.String())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully marshalling a DateTime to JSON
func TestDateTime_MarshalJSON(t *testing.T) {
	scenarios := []struct {
		date     string
		expected string
	}{
		{"", `""`},
		{"2022-01-01 11:23:45.678", `"2022-01-01 11:23:45.678Z"`},
	}

	for i, s := range scenarios {
		dt, err := ParseDateTime(s.date)
		require.Nil(t, err, "(%d) %v", i, err)

		result, err := dt.MarshalJSON()
		require.Nil(t, err, "(%d) %v", i, err)
		require.Equal(t, s.expected, string(result), "(%d) Expected %q, got %q", i, s.expected, string(result))
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDateTime_UnmarshalJSON(t *testing.T) {
	// Test successfully unmarshalling a DateTime from JSON
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			date     string
			expected string
		}{
			{`"2022-01-01 11:23:45.678"`, "2022-01-01 11:23:45.678Z"},
			{`1641024040`, "2022-01-01 08:00:40.000Z"},
		}

		for i, s := range scenarios {
			var dt DateTime
			dt.UnmarshalJSON([]byte(s.date))
			require.Equal(t, s.expected, dt.String(), "(%d) Expected %q, got %q", i, s.expected, dt.String())
		}
	})

	// Test erroring when an invalid JSON is provided
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			date     string
			expected string
		}{
			{"", ""},
			{"invalid_json", ""},
			{"'123'", ""},
			{"2022-01-01 11:23:45.678", ""},
		}

		for _, tt := range tests {
			var dt DateTime
			err := dt.UnmarshalJSON([]byte(tt.date))
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expected)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the value of a DateTime
func TestDateTime_Value(t *testing.T) {
	scenarios := []struct {
		value    any
		expected string
	}{
		{"", ""},
		{"invalid", ""},
		{1641024040, "2022-01-01 08:00:40.000Z"},
		{"2022-01-01 11:23:45.678", "2022-01-01 11:23:45.678Z"},
		{NowDateTime(), NowDateTime().String()},
	}

	for i, s := range scenarios {
		dt, _ := ParseDateTime(s.value)
		result, err := dt.Value()
		require.Nil(t, err, "(%d) %v", i, err)
		require.Equal(t, s.expected, result, "(%d) Expected %q, got %q", i, s.expected, result)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully scanning a DateTime
func TestDateTime_Scan(t *testing.T) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	scenarios := []struct {
		value    any
		expected string
	}{
		{nil, ""},
		{"", ""},
		{"invalid", ""},
		{NowDateTime(), now},
		{time.Now(), now},
		{1.0, ""},
		{1641024040, "2022-01-01 08:00:40.000Z"},
		{"2022-01-01 11:23:45.678", "2022-01-01 11:23:45.678Z"},
	}

	for i, s := range scenarios {
		var dateTime DateTime

		err := dateTime.Scan(s.value)
		require.Nil(t, err, "(%d) %v", i, err)
		require.Contains(t, dateTime.String(), s.expected, "(%d) Expected %q, got %q", i, s.expected, dateTime.String())
	}
}
