package coursemetadata

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestReadMetadata(t *testing.T) {
	t.Run("file exists with valid JSON", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		// Create course directory
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		// Write metadata file
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		data := `{
  "tags": ["go", "programming"]
}`
		require.NoError(t, afero.WriteFile(fs, metadataPath, []byte(data), 0644))

		// Read metadata
		metadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Equal(t, []string{"go", "programming"}, metadata.Tags)
		require.Empty(t, metadata.Description)
	})

	t.Run("file exists with description", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		// Create course directory
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		// Write metadata file with description
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		data := `{
  "description": "A course about Go programming",
  "tags": ["go", "programming"]
}`
		require.NoError(t, afero.WriteFile(fs, metadataPath, []byte(data), 0644))

		// Read metadata
		metadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Equal(t, []string{"go", "programming"}, metadata.Tags)
		require.Equal(t, "A course about Go programming", metadata.Description)
	})

	t.Run("file exists with only description", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		// Create course directory
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		// Write metadata file with only description
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		data := `{
  "description": "A course description"
}`
		require.NoError(t, afero.WriteFile(fs, metadataPath, []byte(data), 0644))

		// Read metadata
		metadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Empty(t, metadata.Tags)
		require.Equal(t, "A course description", metadata.Description)
	})

	t.Run("file exists with empty tags", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadataPath := filepath.Join(coursePath, MetadataFileName)
		data := `{
  "tags": []
}`
		require.NoError(t, afero.WriteFile(fs, metadataPath, []byte(data), 0644))

		metadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Empty(t, metadata.Tags)
	})

	t.Run("file doesn't exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.Nil(t, metadata)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadataPath := filepath.Join(coursePath, MetadataFileName)
		require.NoError(t, afero.WriteFile(fs, metadataPath, []byte("invalid json"), 0644))

		metadata, err := ReadMetadata(fs, coursePath)
		require.Error(t, err)
		require.Nil(t, metadata)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestWriteMetadata(t *testing.T) {
	t.Run("write new file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"go", "programming"},
		}

		err := WriteMetadata(fs, coursePath, metadata)
		require.NoError(t, err)

		// Verify file was created
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		exists, err := afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		require.True(t, exists)

		// Read back and verify
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, metadata.Tags, readMetadata.Tags)
		require.Empty(t, readMetadata.Description)
	})

	t.Run("write new file with description", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "A course about Go programming",
			Tags:        []string{"go", "programming"},
		}

		err := WriteMetadata(fs, coursePath, metadata)
		require.NoError(t, err)

		// Verify file was created
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		exists, err := afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		require.True(t, exists)

		// Read back and verify
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, metadata.Tags, readMetadata.Tags)
		require.Equal(t, metadata.Description, readMetadata.Description)
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		// Write initial metadata
		initialMetadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"old"},
		}
		require.NoError(t, WriteMetadata(fs, coursePath, initialMetadata))

		// Overwrite with new metadata
		newMetadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"new", "tags"},
		}
		require.NoError(t, WriteMetadata(fs, coursePath, newMetadata))

		// Verify
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, []string{"new", "tags"}, readMetadata.Tags)
	})

	t.Run("write empty tags", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{},
		}

		err := WriteMetadata(fs, coursePath, metadata)
		require.NoError(t, err)

		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Empty(t, readMetadata.Tags)
	})

	t.Run("nil metadata", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		err := WriteMetadata(fs, coursePath, nil)
		require.NoError(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDeleteMetadata(t *testing.T) {
	t.Run("delete existing file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"go"},
		}
		require.NoError(t, WriteMetadata(fs, coursePath, metadata))

		// Verify file exists
		metadataPath := filepath.Join(coursePath, MetadataFileName)
		exists, err := afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		require.True(t, exists)

		// Delete
		err = DeleteMetadata(fs, coursePath)
		require.NoError(t, err)

		// Verify file is gone
		exists, err = afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		coursePath := "/test-course"

		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		err := DeleteMetadata(fs, coursePath)
		require.NoError(t, err)
	})
}
