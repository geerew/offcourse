package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_Open(t *testing.T) {
	// Test erroring when opening a file that does not exist
	t.Run("file does not exist", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		res, err := fs.Open("'")

		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
		require.Nil(t, res)
	})

	// Test successfully opening a file that exists
	t.Run("file exists", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Create("/a")

		res, err := fs.Open("/a")
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ReadDir(t *testing.T) {
	// Test erroring when reading a path that does not exist
	t.Run("open error", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		res, err := fs.ReadDir("'", false)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to open path ': open ': file does not exist")
	})

	// Test erroring when reading a path that is not a directory
	t.Run("read path error", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Create("/test")
		res, err := fs.ReadDir("/test", false)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to read path /test: readdir /test: not a dir")
	})

	// Test successfully reading a path that exists
	t.Run("success", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Create("/a")
		fs.Create("/b")
		fs.Mkdir("/c", 0755)

		res, err := fs.ReadDir("/", true)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 2, len(res.Files))
		require.Equal(t, 1, len(res.Directories))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ReadDirFlat(t *testing.T) {
	// Test erroring when opening a path that does not exist
	t.Run("open error", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		res, err := fs.ReadDirFlat("'", 1)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to open path ': open ': file does not exist")
	})

	// Test erroring when reading a path that is not a directory
	t.Run("read path error", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Create("/test")
		res, err := fs.ReadDirFlat("/test", 1)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to read path /test: readdir /test: not a dir")
	})

	// Test successfully reading a path that exists
	t.Run("success", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Mkdir("/1", 0755)
		fs.Mkdir("/2/2", 0755)
		fs.Mkdir("/3/3/3", 0755)
		fs.Mkdir("/4/4/4/4", 0755)
		fs.Create("/f1")
		fs.Create("/1/f1")
		fs.Create("/2/f1")
		fs.Create("/2/2/f1")
		fs.Create("/3/f1")
		fs.Create("/3/3/f1")
		fs.Create("/3/3/3/f1")
		fs.Create("/4/f1")
		fs.Create("/4/4/f1")
		fs.Create("/4/4/4/f1")
		fs.Create("/4/4/4/4/f1")

		res, err := fs.ReadDirFlat("/", 0)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 1, len(res))

		res, err = fs.ReadDirFlat("/", 1)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 1, len(res))

		res, err = fs.ReadDirFlat("/", 2)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 5, len(res))

		res, err = fs.ReadDirFlat("/", 9999)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 11, len(res))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_RemoveAllContents(t *testing.T) {
	// Test erroring when removing a path that does not exist
	t.Run("path does not exist", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		err := fs.RemoveAllContents("/nonexistent")

		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	// Test erroring when removing a path that is not a directory
	t.Run("path is not a directory", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Create("/file")
		err := fs.RemoveAllContents("/file")

		require.Error(t, err)
		require.EqualError(t, err, "path /file is not a directory")
	})

	// Test successfully removing a path that is an empty directory
	t.Run("success empty directory", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.Mkdir("/dir", 0755)
		err := fs.RemoveAllContents("/dir")

		require.NoError(t, err)

		// Directory should still exist but be empty
		contents, err := fs.ReadDir("/dir", false)
		require.NoError(t, err)
		require.Empty(t, contents.Files)
		require.Empty(t, contents.Directories)
	})

	// Test successfully removing a path that is a directory with contents
	t.Run("success directory with contents", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		fs.MkdirAll("/dir/subdir", 0755)
		fs.Create("/dir/file1")
		fs.Create("/dir/file2")
		fs.Create("/dir/subdir/file3")

		err := fs.RemoveAllContents("/dir")
		require.NoError(t, err)

		// Directory should still exist but be empty
		contents, err := fs.ReadDir("/dir", false)
		require.NoError(t, err)
		require.Empty(t, contents.Files)
		require.Empty(t, contents.Directories)

		// Removed paths should no longer exist
		_, err = fs.Stat("/dir/subdir")
		require.True(t, os.IsNotExist(err))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NonWslDrives(t *testing.T) {
	// Test successfully listing drives on a non-WSL system
	t.Run("success", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		drives, err := fs.nonWslDrives()

		require.NoError(t, err)
		require.NotEmpty(t, drives)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_WslDrives(t *testing.T) {
	// Test erroring when opening a path that does not exist
	t.Run("error", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		drives, err := fs.wslDrives()

		require.Nil(t, drives)
		require.EqualError(t, err, "unable to open path /mnt: open /mnt: file does not exist")
	})

	// Test successfully listing drives on a WSL system
	t.Run("success", func(t *testing.T) {
		fs := New(afero.NewMemMapFs())

		// Create WSL directory structure
		paths := []string{"/mnt/c", "/mnt/d", "/mnt/wsl", "/mnt/wslg"}
		for _, p := range paths {
			err := fs.MkdirAll(p, os.ModePerm)
			require.NoError(t, err)
		}

		drives, err := fs.wslDrives()
		require.NoError(t, err)
		require.Len(t, drives, 3)
		require.ElementsMatch(t, []string{"/", filepath.Join("/mnt", "c"), filepath.Join("/mnt", "d")}, drives)
	})
}
