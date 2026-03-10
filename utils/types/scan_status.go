package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/spf13/cast"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ScanStatusType defines the type of scan status
type ScanStatusType string

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	ScanStatusWaiting    ScanStatusType = "waiting"
	ScanStatusProcessing ScanStatusType = "processing"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewScanStatusWaiting creates a ScanStatusType with the status of waiting
func NewScanStatusWaiting() ScanStatusType {
	return ScanStatusWaiting
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewScanStatusProcessing creates a ScanStatusType with the status of processing
func NewScanStatusProcessing() ScanStatusType {
	return ScanStatusProcessing
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsValid checks if the scan status is valid
func (s ScanStatusType) IsValid() bool {
	switch s {
	case ScanStatusWaiting, ScanStatusProcessing:
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsWaiting returns true if the status is waiting
func (s ScanStatusType) IsWaiting() bool {
	return s == ScanStatusWaiting
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsProcessing returns true if the status is processing
func (s ScanStatusType) IsProcessing() bool {
	return s == ScanStatusProcessing
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// String implements the `Stringer` interface
func (s ScanStatusType) String() string {
	return string(s)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MarshalJSON implements the `json.Marshaler` interface
func (s ScanStatusType) MarshalJSON() ([]byte, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid scan status: %s", s)
	}

	return []byte(`"` + string(s) + `"`), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnmarshalJSON implements the `json.Unmarshaler` interface
func (s *ScanStatusType) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	return s.Scan(raw)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Value implements the `driver.Valuer` interface
func (s ScanStatusType) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid scan status: %s", s)
	}

	return string(s), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Scan implements `sql.Scanner` interface
func (s *ScanStatusType) Scan(value any) error {
	vv := cast.ToString(value)

	switch vv {
	case string(ScanStatusWaiting):
		*s = ScanStatusWaiting
	case string(ScanStatusProcessing):
		*s = ScanStatusProcessing
	default:
		return fmt.Errorf("invalid scan status: %s", vv)
	}

	return nil
}
