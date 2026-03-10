package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully creating a new scan status of waiting
func TestScanStatus_NewScanStatusWaiting(t *testing.T) {
	require.Equal(t, ScanStatusWaiting, NewScanStatusWaiting())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully creating a new scan status of processing
func TestScanStatus_NewScanStatusProcessing(t *testing.T) {
	require.Equal(t, ScanStatusProcessing, NewScanStatusProcessing())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if a scan status is waiting
func TestScanStatus_IsWaiting(t *testing.T) {
	require.True(t, NewScanStatusWaiting().IsWaiting())
	require.False(t, NewScanStatusProcessing().IsWaiting())
	require.False(t, ScanStatusType("").IsWaiting())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if a scan status is processing
func TestScanStatus_IsProcessing(t *testing.T) {
	require.False(t, NewScanStatusWaiting().IsProcessing())
	require.True(t, NewScanStatusProcessing().IsProcessing())
	require.False(t, ScanStatusType("").IsProcessing())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if a scan status is valid
func TestScanStatus_IsValid(t *testing.T) {
	require.True(t, NewScanStatusWaiting().IsValid())
	require.True(t, NewScanStatusProcessing().IsValid())
	require.False(t, ScanStatusType("").IsValid())
	require.False(t, ScanStatusType("invalid").IsValid())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanStatus_MarshalJSON(t *testing.T) {
	// Test successfully marshalling a scan status of waiting
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			scanStatus ScanStatusType
			expected   string
		}{
			{NewScanStatusWaiting(), `"waiting"`},
			{NewScanStatusProcessing(), `"processing"`},
		}

		for _, tt := range scenarios {
			res, err := tt.scanStatus.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(res))
		}
	})

	// Test erroring when a scan status is invalid
	t.Run("error", func(t *testing.T) {
		scanStatus := ScanStatusType("invalid")
		_, err := scanStatus.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid scan status")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanStatus_UnmarshalJSON(t *testing.T) {
	// Test successfully unmarshalling a scan status from JSON
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			input    string
			expected ScanStatusType
			err      string
		}{
			{`"waiting"`, ScanStatusWaiting, ""},
			{`"processing"`, ScanStatusProcessing, ""},
		}

		for _, tt := range scenarios {
			var s ScanStatusType
			err := s.UnmarshalJSON([]byte(tt.input))
			require.NoError(t, err)
			require.Equal(t, tt.expected, s)
		}
	})

	// Test erroring when an invalid JSON is provided
	t.Run("error", func(t *testing.T) {
		scenarios := []struct {
			input    string
			expected ScanStatusType
			err      string
		}{
			{`"invalid"`, "", "invalid scan status"},
			{`""`, "", "invalid scan status"},
			{`"bob"`, "", "invalid scan status"},
		}

		for _, tt := range scenarios {
			var s ScanStatusType
			err := s.UnmarshalJSON([]byte(tt.input))
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.err)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the value of a scan status
func TestScanStatus_Value(t *testing.T) {
	// Test successfully getting the value of a scan status
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			scanStatus ScanStatusType
			expected   string
		}{
			{NewScanStatusWaiting(), "waiting"},
			{NewScanStatusProcessing(), "processing"},
		}

		for _, tt := range scenarios {
			res, err := tt.scanStatus.Value()
			require.NoError(t, err)
			require.Equal(t, tt.expected, res)
		}
	})

	// Test erroring when a scan status is invalid
	t.Run("error", func(t *testing.T) {
		scanStatus := ScanStatusType("invalid")
		_, err := scanStatus.Value()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid scan status")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanStatus_Scan(t *testing.T) {
	// Test successfully scanning a scan status from various inputs
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			value    any
			expected ScanStatusType
		}{
			{"waiting", ScanStatusWaiting},
			{"processing", ScanStatusProcessing},
		}

		for _, tt := range tests {
			var s ScanStatusType

			err := s.Scan(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.expected, s)
		}
	})

	// Test erroring when an invalid scan status is provided
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			value any
		}{
			{nil},
			{""},
			{"invalid"},
		}

		for _, tt := range tests {
			var s ScanStatusType

			err := s.Scan(tt.value)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid scan status")
		}
	})
}
