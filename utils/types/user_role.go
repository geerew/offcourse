package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UserRole defines the possible roles for a user
type UserRole string

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Principal represents a user with a specific role
type Principal struct {
	UserID string
	Role   UserRole
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewUserRole creates a new UserRole
//
// Defaults to UserRoleUser if the role provided is invalid
func NewUserRole(role string) UserRole {
	switch role {
	case "admin":
		return UserRoleAdmin
	case "user":
		return UserRoleUser
	default:
		return UserRoleUser
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsValid checks if the role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleUser:
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// String implements the Stringer interface
func (r UserRole) String() string {
	return string(r)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MarshalJSON implements the `json.Marshaler` interface
func (r UserRole) MarshalJSON() ([]byte, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", r)
	}

	return json.Marshal(string(r))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnmarshalJSON implements the `json.Unmarshaler` interface
func (r *UserRole) UnmarshalJSON(data []byte) error {
	var role string
	if err := json.Unmarshal(data, &role); err != nil {
		return err
	}

	userRole := UserRole(role)
	if !userRole.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	*r = userRole
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Value implements the `driver.Valuer` interface
func (r UserRole) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", r)
	}
	return string(r), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Scan implements the sql.Scanner interface
func (r *UserRole) Scan(value interface{}) error {
	role, ok := value.(string)
	if !ok {
		return errors.New("invalid data type for UserRole")
	}

	userRole := UserRole(role)
	if !userRole.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	*r = userRole
	return nil
}
