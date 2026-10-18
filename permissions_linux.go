//go:build linux

package main

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func (pkg *Package) checkPermissions(removing bool) error {
	if os.Geteuid() == 0 {
		return nil
	}

	for index := range len(pkg.Definition.Companions) + 1 {
		path := pkg.BinaryPath(index)

		switch pkg.Definition.Source {
		case "go":
			path = filepath.Dir(filepath.Dir(path))
		case "zig":
			path = filepath.Dir(path)
		}

		if removing {
			_, err := os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
		}

		err := checkDestinationPermissions(path)
		if err != nil {
			return err
		}
	}

	if len(pkg.Definition.Variants) == 0 {
		return nil
	}

	if removing {
		config, err := ReadConfig()
		if err != nil {
			return err
		}

		if _, exists := config.Packages[pkg.Definition.Name]; !exists {
			return nil
		}
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	return checkDestinationPermissions(path)
}

// Files are staged and renamed, so access to the parent matters, not the file's write bits.
func checkDestinationPermissions(path string) error {
	if os.Geteuid() == 0 {
		return nil
	}

	directory := filepath.Dir(path)

	for {
		err := unix.Faccessat(unix.AT_FDCWD, directory, unix.W_OK|unix.X_OK, unix.AT_EACCESS)
		if errors.Is(err, os.ErrNotExist) {
			parent := filepath.Dir(directory)
			if parent != directory {
				directory = parent

				continue
			}
		}

		if errors.Is(err, os.ErrPermission) {
			return preflightPermissionError(directory, err)
		}

		if err != nil {
			// Unsupported checks and non-permission failures are left to the operation itself.
			return nil
		}

		break
	}

	var parent unix.Stat_t

	err := unix.Stat(directory, &parent)
	if err != nil || parent.Mode&unix.S_ISVTX == 0 || parent.Uid == uint32(os.Geteuid()) {
		return nil
	}

	var target unix.Stat_t

	err = unix.Lstat(path, &target)
	if err == nil && target.Uid != uint32(os.Geteuid()) {
		return preflightPermissionError(path, unix.EPERM)
	}

	return nil
}

func preflightPermissionError(path string, cause error) error {
	return &writePermissionError{
		cause: &os.PathError{
			Op:   "write access required",
			Path: path,
			Err:  cause,
		},
		preflight: true,
	}
}
