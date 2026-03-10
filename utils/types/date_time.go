package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cast"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DefaultDateLayout defines the default date string layout
const DefaultDateLayout = "2006-01-02 15:04:05.000Z"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DateTime represents a serializable `time.Time`
type DateTime time.Time

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NowDateTime returns new DateTime instance with the current local time
func NowDateTime() DateTime {
	return DateTime(time.Now())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ParseDateTime creates a new DateTime from the provided value
func ParseDateTime(value any) (DateTime, error) {
	var d DateTime
	err := d.Scan(value)
	return d, err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsZero checks whether DateTime has zero time value
func (d DateTime) IsZero() bool {
	return time.Time(d).IsZero()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Equal checks if two DateTime are the same, ignoring milliseconds
func (d DateTime) Equal(other DateTime) bool {
	return time.Time(d).UTC().Truncate(time.Millisecond).Equal(time.Time(other).UTC().Truncate(time.Millisecond))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// String implements the `Stringer` interface
func (d DateTime) String() string {
	t := time.Time(d)

	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(DefaultDateLayout)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MarshalJSON implements the `json.Marshaler` interface
func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnmarshalJSON implements the `json.Unmarshaler` interface
func (d *DateTime) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err == nil {
		return d.Scan(raw)
	}

	var num json.Number
	if err := json.Unmarshal(b, &num); err == nil {
		i, err := num.Int64()
		if err != nil {
			return err
		}
		return d.Scan(i)
	}

	return fmt.Errorf("invalid datetime json: %s", string(b))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Value implements the `driver.Valuer` interface
func (d DateTime) Value() (driver.Value, error) {
	return d.String(), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Scan implements `sql.Scanner` interface, scanning the provided value into
// the DateTime
func (d *DateTime) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		*d = DateTime(v)
	case DateTime:
		*d = v
	case string:
		if v == "" {
			*d = DateTime(time.Time{})
		} else {
			t, err := time.Parse(DefaultDateLayout, v)
			if err != nil {
				t = cast.ToTime(v)
			}

			*d = DateTime(t)
		}
	case int, int64, int32, uint, uint64, uint32:
		*d = DateTime(cast.ToTime(v))
	default:
		str := cast.ToString(v)
		if str == "" {
			*d = DateTime(time.Time{})
		} else {
			*d = DateTime(cast.ToTime(str))
		}
	}

	return nil
}
