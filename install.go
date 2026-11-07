package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coalaura/semver"
)

func (pkg *Package) Install(version semver.SemVer) error {
	switch pkg.Definition.Source {
	case "go":
		return InstallGo(version)
	case "zig":
		return InstallZig(version)
	case "pace":
		err := ValidatePaceGoVersion(version)
		if err != nil {
			return err
		}
	}

	asset := pkg.Expand(pkg.Target.Asset, version.String())

	path, err := pkg.Download(version)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	if pkg.Definition.Source == "msi" {
		return pkg.InstallMSI(path)
	}

	directory, err := os.MkdirTemp("", "tap-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(directory)

	source, err := pkg.Extract(path, directory, asset, version.String())
	if err != nil {
		return err
	}

	sources := make([]string, len(pkg.Definition.Companions)+1)
	sources[0] = source

	for index := range pkg.Definition.Companions {
		sources[index+1], err = FindFile(directory, filepath.Base(pkg.BinaryPath(index+1)))
		if err != nil {
			return err
		}
	}

	err = pkg.PrepareDirectory()
	if err != nil {
		return err
	}

	// Validate on the destination filesystem so noexec temp mounts do not prevent installation.
	stagingDirectory, err := os.MkdirTemp(filepath.Dir(pkg.Path), ".tap-install-*")
	if err != nil {
		return writeError(err)
	}

	defer os.RemoveAll(stagingDirectory)

	for index, source := range sources {
		staged := filepath.Join(stagingDirectory, filepath.Base(pkg.BinaryPath(index)))

		err = CopyFileMode(source, staged, 0755)
		if err != nil {
			return err
		}

		sources[index] = staged
	}

	// Companions share the verified archive; only the primary binary is the version probe.
	err = pkg.ValidateBinary(sources[0], version)
	if err != nil {
		return err
	}

	// Install the version-probe binary last so an interrupted update is retried.
	for index, source := range slices.Backward(sources) {
		destination := pkg.BinaryPath(index)

		err = ReplaceFile(source, destination)
		if err != nil {
			return writeError(err)
		}

		if strings.HasPrefix(pkg.Target.Path, "{home}/") {
			err = ChownToInvokingUser(destination)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (pkg *Package) Extract(path, directory, asset, version string) (string, error) {
	var err error

	switch {
	case strings.HasSuffix(asset, ".tar.gz"):
		err = ExtractTarGzFile(path, directory)
	case strings.HasSuffix(asset, ".tar.xz"):
		err = ExtractTarXzFile(path, directory)
	case strings.HasSuffix(asset, ".zip"):
		err = ExtractZipFile(path, directory)
	default:
		return path, nil
	}

	if err != nil {
		return "", err
	}

	if pkg.Target.Executable != "" {
		return ArchivePath(directory, pkg.Expand(pkg.Target.Executable, version))
	}

	return FindFile(directory, pkg.Expand("{binary}{ext}", version))
}

func (pkg *Package) PrepareDirectory() error {
	directory := filepath.Dir(pkg.Path)

	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return writeError(err)
	}

	if !strings.HasPrefix(pkg.Target.Path, "{home}/") {
		return nil
	}

	home, err := UserHomeDir()
	if err != nil {
		return err
	}

	relative, err := filepath.Rel(home, directory)
	if err != nil {
		return err
	}

	if !filepath.IsLocal(relative) {
		return errors.New("user install path is outside the home directory")
	}

	for directory != filepath.Clean(home) {
		err = ChownToInvokingUser(directory)
		if err != nil {
			return writeError(err)
		}

		directory = filepath.Dir(directory)
	}

	return nil
}
