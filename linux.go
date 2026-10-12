//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coalaura/semver"
)

func (pkg *Package) InstallMSI(path string) error {
	return fmt.Errorf("%s requires windows installer", pkg.Definition.Name)
}

func (pkg *Package) RemoveMSI() error {
	return fmt.Errorf("%s requires windows installer", pkg.Definition.Name)
}

func InstallGo(ver semver.SemVer) error {
	path, err := DownloadGoFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("/usr/local", ".tap-go-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractTarGzFile(path, dir)
	if err != nil {
		return err
	}

	err = ValidateBinary(filepath.Join(dir, "go", "bin", "go"), []string{"version"}, ver)
	if err != nil {
		return err
	}

	return ReplaceDirectory(filepath.Join(dir, "go"), "/usr/local/go")
}

func InstallZig(ver semver.SemVer) error {
	path, err := DownloadZigFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("/usr/local", ".tap-zig-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractTarXzFile(path, dir)
	if err != nil {
		return err
	}

	payload := filepath.Join(dir, fmt.Sprintf("zig-x86_64-linux-%s", ver.String()))

	err = ValidateBinary(filepath.Join(payload, "zig"), []string{"version"}, ver)
	if err != nil {
		return err
	}

	return ReplaceDirectory(payload, "/usr/local/zig")
}

func RemoveGo() error {
	return os.RemoveAll("/usr/local/go")
}
