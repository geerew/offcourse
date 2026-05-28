package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/geerew/off-course/utils"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// FS represents the application filesystem
type FS struct {
	afero.Fs
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New create a new filesystem
func New(backend afero.Fs) *FS {
	return &FS{
		Fs: backend,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PathContents defines the fields populated during a path
// scan
type PathContents struct {
	Files       []fs.FileInfo
	Directories []fs.FileInfo
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Open opens a file with a given name
func (f *FS) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}

		return nil, err
	}

	return file, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ReadDir reads the contents of a path, building a slice of files and directories
func (f FS) ReadDir(path string, sortResult bool) (*PathContents, error) {
	path = utils.NormalizeWindowsDrive(path)

	items, err := f.pathItems(path)
	if err != nil {
		return nil, err
	}

	if sortResult {
		sort.Strings(items)
	}

	directories := make([]fs.FileInfo, 0)
	files := make([]fs.FileInfo, 0)

	for _, file := range items {
		fullPath := filepath.Join(path, file)

		if fileStat, err := f.Stat(fullPath); err == nil {
			if fileStat.IsDir() {
				directories = append(directories, fileStat)
			} else {
				files = append(files, fileStat)
			}
		}
	}

	return &PathContents{
		Files:       files,
		Directories: directories,
	}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ReadDirFlat recursively reads a directory down to a certain depth, building a slice of
// paths
//
// It is a wrapper around `recursivelyReadDir` that normalizes the path and forces the current
// depth to 0
func (f FS) ReadDirFlat(path string, depth int) ([]string, error) {
	return f.recursivelyReadDir(utils.NormalizeWindowsDrive(path), depth, 0)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AvailableDrives returns a string slice of available drives on this system
//
// For non-WSL systems, `gopsutil` is used. For WSL systems, the string slice is generated
// manually
func (f FS) AvailableDrives() ([]string, error) {
	kernel, err := host.KernelVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to lookup system information: %w", err)
	}

	if strings.Contains(strings.ToLower(kernel), "wsl") {
		return f.wslDrives()
	}

	return f.nonWslDrives()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RemoveAllContents removes all everything (files and directories) within a given path
func (f FS) RemoveAllContents(path string) error {
	path = utils.NormalizeWindowsDrive(path)

	fileInfo, err := f.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &fs.PathError{Op: "removeall", Path: path, Err: fs.ErrNotExist}
		}
		return fmt.Errorf("unable to stat path %s: %w", path, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}

	items, err := f.pathItems(path)
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %w", path, err)
	}

	for _, item := range items {
		fullPath := filepath.Join(path, item)

		if err := f.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("unable to remove %s: %w", fullPath, err)
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// pathItems does the common work of opening a path and listing its contents
func (f FS) pathItems(path string) ([]string, error) {
	file, err := f.Fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open path %s: %w", path, err)
	}

	items, err := file.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("unable to read path %s: %w", path, err)
	}

	return items, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// recursivelyReadDir does the common work of recursively reading a directory down to a
// certain depth, building a slice of paths
func (f FS) recursivelyReadDir(path string, maxDepth, currDepth int) ([]string, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}

	if currDepth == maxDepth {
		return nil, nil
	}

	res := []string{}

	items, err := f.pathItems(path)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		fullPath := filepath.Join(path, item)

		if fileStat, err := f.Stat(fullPath); err == nil {
			if fileStat.IsDir() {
				recursiveRes, err := f.recursivelyReadDir(fullPath, maxDepth, currDepth+1)
				if err != nil {
					return nil, err
				}

				res = append(res, recursiveRes...)
			} else {
				res = append(res, fullPath)
			}
		}
	}

	return res, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// nonWslDrives builds a list of available drives for non-wsl systems via `gopsutil`
//
// Ignores the FS instance and only queries the real disk partitions
func (f FS) nonWslDrives() ([]string, error) {
	var drives []string

	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list drives: %w", err)
	}

	for _, partition := range partitions {
		drives = append(drives, partition.Mountpoint)
	}

	return drives, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// wslDrives manually builds a list of available drives in WSL
func (f FS) wslDrives() ([]string, error) {
	drives := []string{"/"}

	items, err := f.ReadDir("/mnt", true)
	if err != nil {
		return nil, err
	}

	for _, directory := range items.Directories {
		if !strings.Contains(directory.Name(), "wsl") {
			drives = append(drives, filepath.Join("/mnt", directory.Name()))
		}
	}

	return drives, nil
}
