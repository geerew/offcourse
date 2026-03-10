package types

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestUserRole_NewUserRole(t *testing.T) {
	// Test successfully creating a new user role
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			input    string
			expected UserRole
		}{
			{"admin", UserRoleAdmin},
			{"user", UserRoleUser},
		}

		for i, tt := range scenarios {
			result := NewUserRole(tt.input)
			assert.Equal(t, tt.expected, result, "(%d) Expected %s, got %s", i, tt.expected, result)
		}
	})

	// Test erroring when an invalid user role is provided
	t.Run("error", func(t *testing.T) {
		scenarios := []struct {
			input    string
			expected UserRole
		}{
			{"invalid", UserRoleUser},
			{"", UserRoleUser},
			{"ADMIN", UserRoleUser},
		}

		for i, tt := range scenarios {
			result := NewUserRole(tt.input)
			assert.Equal(t, tt.expected, result, "(%d) Expected %s, got %s", i, tt.expected, result)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the string representation of a user role
func TestUserRole_String(t *testing.T) {
	assert.Equal(t, "admin", UserRoleAdmin.String())
	assert.Equal(t, "user", UserRoleUser.String())
	assert.Equal(t, "invalid", UserRole("invalid").String())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully checking if a user role is valid
func TestUserRole_IsValid(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected bool
	}{
		{UserRoleAdmin, true},
		{UserRoleUser, true},
		{UserRole("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.IsValid(), "(%s) Expected %t, got %t", string(tt.role), tt.expected, tt.role.IsValid())
		})
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestUserRole_MarshalJSON(t *testing.T) {
	// Test successfully marshalling a user role to JSON
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			role     UserRole
			expected string
		}{
			{UserRoleAdmin, `"admin"`},
			{UserRoleUser, `"user"`},
		}

		for _, tt := range scenarios {
			data, err := tt.role.MarshalJSON()
			assert.NoError(t, err, "(%s) Expected no error, got %v", string(tt.role), err)
			assert.Equal(t, tt.expected, string(data), "(%s) Expected %s, got %s", string(tt.role), tt.expected, string(data))
		}
	})

	// Test erroring when a user role is invalid
	t.Run("error", func(t *testing.T) {

		invalid := UserRole("invalid")
		_, err := invalid.MarshalJSON()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user role")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestUserRole_UnmarshalJSON(t *testing.T) {
	// Test successfully unmarshalling a user role from JSON
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			data     string
			expected UserRole
		}{
			{`"admin"`, UserRoleAdmin},
			{`"user"`, UserRoleUser},
		}

		for _, tt := range scenarios {
			var role UserRole
			err := role.UnmarshalJSON([]byte(tt.data))
			assert.NoError(t, err, "(%s) Expected no error, got %v", tt.data, err)
			assert.Equal(t, tt.expected, role, "(%s) Expected %s, got %s", tt.data, tt.expected, role)
		}
	})

	// Test erroring when an invalid JSON is provided
	t.Run("error", func(t *testing.T) {
		scenarios := []struct {
			data string
			err  string
		}{
			{`"invalid"`, "invalid user role"},
			{`""`, "invalid user role"},
			{`"bob"`, "invalid user role"},
		}
		for _, tt := range scenarios {
			var role UserRole
			err := role.UnmarshalJSON([]byte(tt.data))
			assert.Error(t, err, "(%s) Expected error, got %v", tt.data, err)
			assert.Contains(t, err.Error(), tt.err, "(%s) Expected error to contain %s, got %s", tt.data, tt.err, err.Error())
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestUserRole_Value(t *testing.T) {
	// Test successfully getting the value of a user role
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			role     UserRole
			expected driver.Value
		}{
			{UserRoleAdmin, "admin"},
			{UserRoleUser, "user"},
		}
		for _, tt := range scenarios {
			value, err := tt.role.Value()
			assert.NoError(t, err, "(%s) Expected no error, got %v", tt.role, err)
			assert.Equal(t, tt.expected, value, "(%s) Expected %s, got %s", tt.role, tt.expected, value)
		}
	})

	// Test erroring when a user role is invalid
	t.Run("error", func(t *testing.T) {
		invalid := UserRole("invalid")
		_, err := invalid.Value()
		assert.Error(t, err, "(%s) Expected error, got %v", invalid, err)
		assert.Contains(t, err.Error(), "invalid user role", "(%s) Expected error to contain %s, got %s", invalid, "invalid user role", err.Error())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestUserRole_Scan(t *testing.T) {
	// Test successfully scanning a user role from various inputs
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			input    interface{}
			expected UserRole
		}{
			{"admin", UserRoleAdmin},
			{"user", UserRoleUser},
		}
		for _, tt := range scenarios {
			var role UserRole
			err := role.Scan(tt.input)
			assert.NoError(t, err, "(%s) Expected no error, got %v", tt.input, err)
			assert.Equal(t, tt.expected, role, "(%s) Expected %s, got %s", tt.input, tt.expected, role)
		}
	})

	// Test erroring when an invalid input is provided
	t.Run("error", func(t *testing.T) {
		scenarios := []struct {
			input interface{}
			err   string
		}{
			{"invalid", "invalid user role: invalid"},
			{"", "invalid user role: "},
			{"123", "invalid user role: 123"},
			{true, "invalid data type for UserRole"},
			{123.45, "invalid data type for UserRole"},
			{[]byte("admin"), "invalid data type for UserRole"},
		}
		for _, tt := range scenarios {
			var role UserRole
			err := role.Scan(tt.input)
			assert.Error(t, err, "(%s) Expected error, got %v", tt.input, err)
			assert.Equal(t, tt.err, err.Error(), "(%s) Expected error to be %s, got %s", tt.input, tt.err, err.Error())
		}
	})
}
