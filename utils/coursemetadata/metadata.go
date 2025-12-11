package coursemetadata

import (
	"encoding/json"
	"path/filepath"

	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	// MetadataFileName is the name of the course metadata file
	MetadataFileName = "oc.json"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseMetadata represents the metadata stored in oc.json
type CourseMetadata struct {
	Tags []string `json:"tags,omitempty"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ReadMetadata reads the oc.json file from the course root directory.
// Returns nil metadata (not an error) if the file doesn't exist.
func ReadMetadata(fs afero.Fs, coursePath string) (*CourseMetadata, error) {
	metadataPath := filepath.Join(coursePath, MetadataFileName)

	// Check if file exists
	exists, err := afero.Exists(fs, metadataPath)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	// Read file
	data, err := afero.ReadFile(fs, metadataPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var metadata CourseMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WriteMetadata writes the oc.json file to the course root directory.
// Creates the file if it doesn't exist, overwrites if it does.
func WriteMetadata(fs afero.Fs, coursePath string, metadata *CourseMetadata) error {
	if metadata == nil {
		return nil
	}

	metadataPath := filepath.Join(coursePath, MetadataFileName)

	// Ensure directory exists
	dir := filepath.Dir(metadataPath)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal JSON with indentation
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	// Add newline at end of file
	data = append(data, '\n')

	// Write file
	if err := afero.WriteFile(fs, metadataPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteMetadata deletes the oc.json file from the course root directory.
// Returns nil if the file doesn't exist (not an error).
func DeleteMetadata(fs afero.Fs, coursePath string) error {
	metadataPath := filepath.Join(coursePath, MetadataFileName)

	exists, err := afero.Exists(fs, metadataPath)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return fs.Remove(metadataPath)
}
