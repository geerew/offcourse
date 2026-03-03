package appfs

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
		appFs := New(afero.NewMemMapFs())

		res, err := appFs.Open("'")

		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
		require.Nil(t, res)
	})

	// Test successfully opening a file that exists
	t.Run("file exists", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Create("/a")

		res, err := appFs.Open("/a")
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ReadDir(t *testing.T) {
	// Test erroring when reading a path that does not exist
	t.Run("open error", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		res, err := appFs.ReadDir("'", false)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to open path ': open ': file does not exist")
	})

	// Test erroring when reading a path that is not a directory
	t.Run("read path error", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Create("/test")
		res, err := appFs.ReadDir("/test", false)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to read path /test: readdir /test: not a dir")
	})

	// Test successfully reading a path that exists
	t.Run("success", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Create("/a")
		appFs.Fs.Create("/b")
		appFs.Fs.Mkdir("/c", 0755)

		res, err := appFs.ReadDir("/", true)
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
		appFs := New(afero.NewMemMapFs())

		res, err := appFs.ReadDirFlat("'", 1)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to open path ': open ': file does not exist")
	})

	// Test erroring when reading a path that is not a directory
	t.Run("read path error", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Create("/test")
		res, err := appFs.ReadDirFlat("/test", 1)

		require.Nil(t, res)
		require.EqualError(t, err, "unable to read path /test: readdir /test: not a dir")
	})

	// Test successfully reading a path that exists
	t.Run("success", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Mkdir("/1", 0755)
		appFs.Fs.Mkdir("/2/2", 0755)
		appFs.Fs.Mkdir("/3/3/3", 0755)
		appFs.Fs.Mkdir("/4/4/4/4", 0755)
		appFs.Fs.Create("/f1")
		appFs.Fs.Create("/1/f1")
		appFs.Fs.Create("/2/f1")
		appFs.Fs.Create("/2/2/f1")
		appFs.Fs.Create("/3/f1")
		appFs.Fs.Create("/3/3/f1")
		appFs.Fs.Create("/3/3/3/f1")
		appFs.Fs.Create("/4/f1")
		appFs.Fs.Create("/4/4/f1")
		appFs.Fs.Create("/4/4/4/f1")
		appFs.Fs.Create("/4/4/4/4/f1")

		res, err := appFs.ReadDirFlat("/", 0)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 1, len(res))

		res, err = appFs.ReadDirFlat("/", 1)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 1, len(res))

		res, err = appFs.ReadDirFlat("/", 2)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 5, len(res))

		res, err = appFs.ReadDirFlat("/", 9999)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 11, len(res))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_RemoveAllContents(t *testing.T) {
	// Test erroring when removing a path that does not exist
	t.Run("path does not exist", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		err := appFs.RemoveAllContents("/nonexistent")

		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	// Test erroring when removing a path that is not a directory
	t.Run("path is not a directory", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Create("/file")
		err := appFs.RemoveAllContents("/file")

		require.Error(t, err)
		require.EqualError(t, err, "path /file is not a directory")
	})

	// Test successfully removing a path that is an empty directory
	t.Run("success empty directory", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.Mkdir("/dir", 0755)
		err := appFs.RemoveAllContents("/dir")

		require.NoError(t, err)

		// Directory should still exist but be empty
		contents, err := appFs.ReadDir("/dir", false)
		require.NoError(t, err)
		require.Empty(t, contents.Files)
		require.Empty(t, contents.Directories)
	})

	// Test successfully removing a path that is a directory with contents
	t.Run("success directory with contents", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		appFs.Fs.MkdirAll("/dir/subdir", 0755)
		appFs.Fs.Create("/dir/file1")
		appFs.Fs.Create("/dir/file2")
		appFs.Fs.Create("/dir/subdir/file3")

		err := appFs.RemoveAllContents("/dir")
		require.NoError(t, err)

		// Directory should still exist but be empty
		contents, err := appFs.ReadDir("/dir", false)
		require.NoError(t, err)
		require.Empty(t, contents.Files)
		require.Empty(t, contents.Directories)

		// Removed paths should no longer exist
		_, err = appFs.Fs.Stat("/dir/subdir")
		require.True(t, os.IsNotExist(err))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NonWslDrives(t *testing.T) {
	// Test successfully listing drives on a non-WSL system
	t.Run("success", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		drives, err := appFs.nonWslDrives()

		require.NoError(t, err)
		require.NotEmpty(t, drives)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_WslDrives(t *testing.T) {
	// Test erroring when opening a path that does not exist
	t.Run("error", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		drives, err := appFs.wslDrives()

		require.Nil(t, drives)
		require.EqualError(t, err, "unable to open path /mnt: open /mnt: file does not exist")
	})

	// Test successfully listing drives on a WSL system
	t.Run("success", func(t *testing.T) {
		appFs := New(afero.NewMemMapFs())

		// Create WSL directory structure
		paths := []string{"/mnt/c", "/mnt/d", "/mnt/wsl", "/mnt/wslg"}
		for _, p := range paths {
			err := appFs.Fs.MkdirAll(p, os.ModePerm)
			require.NoError(t, err)
		}

		drives, err := appFs.wslDrives()
		require.NoError(t, err)
		require.Len(t, drives, 3)
		require.ElementsMatch(t, []string{"/", filepath.Join("/mnt", "c"), filepath.Join("/mnt", "d")}, drives)
	})
}
