//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/coalaura/semver"
)

var msiMutex sync.Mutex

func (pkg *Package) InstallMSI(path string) error {
	return RunMSI("/i", path)
}

func (pkg *Package) RemoveMSI() error {
	version, err := ResolveBinaryVersion(pkg.Path, pkg.Definition.Args)
	if err != nil {
		return err
	}

	path, err := pkg.Download(version)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	return RunMSI("/x", path)
}

func InstallGo(ver semver.SemVer) error {
	path, err := DownloadGoFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	return RunMSI("/i", path)
}

func InstallZig(ver semver.SemVer) error {
	path, err := DownloadZigFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dstDir := filepath.Dir(GetZigBinaryPath())
	parent := filepath.Dir(dstDir)

	dir, err := os.MkdirTemp(parent, ".tap-zig-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractZipFile(path, dir)
	if err != nil {
		return err
	}

	srcDir := filepath.Join(dir, fmt.Sprintf("zig-x86_64-windows-%s", ver.String()))

	err = ValidateBinary(filepath.Join(srcDir, "zig.exe"), []string{"version"}, ver)
	if err != nil {
		return err
	}

	return ReplaceDirectory(srcDir, dstDir)
}

func RemoveGo() error {
	version, err := ResolveBinaryVersion(GetGoBinaryPath(), []string{"version"})
	if err != nil {
		return err
	}

	// Use the matching MSI so Windows Installer also removes its registration and PATH entries.
	path, err := DownloadGoFile(version)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	return RunMSI("/x", path)
}

func RunMSI(action, path string) error {
	// Windows Installer only permits one installation transaction at a time.
	msiMutex.Lock()
	defer msiMutex.Unlock()

	err := RunCommandOrError("msiexec.exe", action, path, "/qn", "/norestart")

	var exitError *exec.ExitError

	if errors.As(err, &exitError) && exitError.ExitCode() == 3010 {
		log.Warnln("done; restart required")

		return nil
	}

	return err
}
