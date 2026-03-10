package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetType defines the type of asset
type AssetType string

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	AssetVideo    AssetType = "video"
	AssetPDF      AssetType = "pdf"
	AssetMarkdown AssetType = "markdown"
	AssetText     AssetType = "text"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewAsset creates an AssetType based upon an extension
//
// Returns an error if the extension is unknown
func NewAsset(ext string) (AssetType, error) {
	switch strings.ToLower(ext) {
	case "avi",
		"mkv",
		"flac",
		"mp4",
		"m4a",
		"mp3",
		"ogv",
		"ogm",
		"ogg",
		"oga",
		"opus",
		"webm",
		"wav":
		return AssetVideo, nil
	case "pdf":
		return AssetPDF, nil
	case "md":
		return AssetMarkdown, nil
	case "txt":
		return AssetText, nil
	default:
		return "", fmt.Errorf("invalid asset extension: %s", ext)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MustAsset creates an AssetType from an extension, panicking on error
func MustAsset(ext string) AssetType {
	at, err := NewAsset(ext)
	if err != nil {
		panic(fmt.Sprintf("MustAsset(%q): %v", ext, err))
	}
	return at
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsValid checks if the asset type is valid
func (a AssetType) IsValid() bool {
	switch a {
	case AssetVideo, AssetPDF, AssetMarkdown, AssetText:
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsVideo returns true if the asset is of type video
func (a AssetType) IsVideo() bool {
	return a == AssetVideo
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsPDF returns true if the asset is of type PDF
func (a AssetType) IsPDF() bool {
	return a == AssetPDF
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsMarkdown returns true if the asset is of type Markdown
func (a AssetType) IsMarkdown() bool {
	return a == AssetMarkdown
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsText returns true if the asset is of type Text
func (a AssetType) IsText() bool {
	return a == AssetText
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// String implements the `Stringer` interface
func (a AssetType) String() string {
	return string(a)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MarshalJSON implements the `json.Marshaler` interface
func (a AssetType) MarshalJSON() ([]byte, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", a)
	}

	return []byte(`"` + string(a) + `"`), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnmarshalJSON implements the `json.Unmarshaler` interface
func (a *AssetType) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	return a.Scan(raw)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Value implements the `driver.Valuer` interface
func (a AssetType) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", a)
	}

	return string(a), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Scan implements `sql.Scanner` interface, scanning the provided value into
// the AssetType
func (a *AssetType) Scan(value any) error {
	vv := cast.ToString(value)

	switch vv {
	case string(AssetVideo):
		*a = AssetVideo
	case string(AssetPDF):
		*a = AssetPDF
	case string(AssetMarkdown):
		*a = AssetMarkdown
	case string(AssetText):
		*a = AssetText
	default:
		return fmt.Errorf("invalid asset type: %s", vv)
	}

	return nil
}
