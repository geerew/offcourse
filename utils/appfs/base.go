package appfs

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

// AppFs represents the application filesystem
type AppFs struct {
	Fs afero.Fs
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New create a new filesystem
func New(fs afero.Fs) *AppFs {
	return &AppFs{
		Fs: fs,
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
func (appFs *AppFs) Open(name string) (fs.File, error) {
	file, err := appFs.Fs.Open(name)
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
func (appFs AppFs) ReadDir(path string, sortResult bool) (*PathContents, error) {
	path = utils.NormalizeWindowsDrive(path)

	items, err := appFs.pathItems(path)
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

		if fileStat, err := appFs.Fs.Stat(fullPath); err == nil {
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
func (appFs AppFs) ReadDirFlat(path string, depth int) ([]string, error) {
	return appFs.recursivelyReadDir(utils.NormalizeWindowsDrive(path), depth, 0)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AvailableDrives returns a string slice of available drives on this system
//
// For non-WSL systems, `gopsutil` is used. For WSL systems, the string slice is generated
// manually
func (appFs AppFs) AvailableDrives() ([]string, error) {
	kernel, err := host.KernelVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to lookup system information: %w", err)
	}

	if strings.Contains(strings.ToLower(kernel), "wsl") {
		return appFs.wslDrives()
	}

	return appFs.nonWslDrives()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RemoveAllContents removes all everything (files and directories) within a given path
func (appFs AppFs) RemoveAllContents(path string) error {
	path = utils.NormalizeWindowsDrive(path)

	fileInfo, err := appFs.Fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &fs.PathError{Op: "removeall", Path: path, Err: fs.ErrNotExist}
		}
		return fmt.Errorf("unable to stat path %s: %w", path, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}

	items, err := appFs.pathItems(path)
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %w", path, err)
	}

	for _, item := range items {
		fullPath := filepath.Join(path, item)

		if err := appFs.Fs.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("unable to remove %s: %w", fullPath, err)
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// pathItems does the common work of opening a path and listing its contents
func (appFs AppFs) pathItems(path string) ([]string, error) {
	f, err := appFs.Fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open path %s: %w", path, err)
	}

	items, err := f.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("unable to read path %s: %w", path, err)
	}

	return items, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// recursivelyReadDir does the common work of recursively reading a directory down to a
// certain depth, building a slice of paths
func (appFs AppFs) recursivelyReadDir(path string, maxDepth, currDepth int) ([]string, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}

	if currDepth == maxDepth {
		return nil, nil
	}

	res := []string{}

	items, err := appFs.pathItems(path)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		fullPath := filepath.Join(path, item)

		if fileStat, err := appFs.Fs.Stat(fullPath); err == nil {
			if fileStat.IsDir() {
				recursiveRes, err := appFs.recursivelyReadDir(fullPath, maxDepth, currDepth+1)
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
// Ignores the appFs filesystem and only queries the real disk partitions
func (appFs AppFs) nonWslDrives() ([]string, error) {
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
func (appFs AppFs) wslDrives() ([]string, error) {
	drives := []string{"/"}

	items, err := appFs.ReadDir("/mnt", true)
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
