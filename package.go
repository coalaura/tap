package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/coalaura/semver"
	"github.com/coalaura/tap/internal/catalog"
)

type Package struct {
	Definition *catalog.Package
	Target     catalog.Target
	Path       string
}

func (pkg *Package) Get() error {
	log.Infof("checking %s\n", pkg.Definition.Name)

	checkStarted := time.Now()

	local, err := pkg.ResolveCurrentVersion()
	if err != nil {
		return err
	}

	remote, err := pkg.FetchLatestVersion()
	if err != nil {
		return err
	}

	if !local.Equal(semver.NewEmptySemVer()) && !remote.HigherThan(local) {
		log.Subf("up to date: %s (checked in %s)\n", local, time.Since(checkStarted).Round(time.Millisecond))

		return nil
	}

	log.Subf("latest %s (checked in %s)\n", remote, time.Since(checkStarted).Round(time.Millisecond))
	log.Stepf("installing %s %s\n", pkg.Definition.Name, remote)

	installStarted := time.Now()

	err = pkg.Install(remote)
	if err != nil {
		return err
	}

	err = pkg.ValidateBinary(pkg.Path, remote)
	if err != nil {
		return err
	}

	log.Successf("installed %s %s in %s\n", pkg.Definition.Name, remote, time.Since(installStarted).Round(time.Millisecond))

	return nil
}

func (pkg *Package) IsInstalled() (bool, error) {
	var installed bool

	for index := range len(pkg.Definition.Companions) + 1 {
		path := pkg.BinaryPath(index)

		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return false, err
		}

		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("%s is not a regular file", path)
		}

		installed = true
	}

	return installed, nil
}

func (pkg *Package) Remove() error {
	installed, err := pkg.IsInstalled()
	if err != nil {
		return err
	}

	if !installed {
		log.Infof("%s: not installed\n", pkg.Definition.Name)

		return nil
	}

	log.Stepf("removing %s\n", pkg.Definition.Name)

	removeStarted := time.Now()

	switch pkg.Definition.Source {
	case "go":
		err = RemoveGo()
	case "zig":
		err = os.RemoveAll(filepath.Dir(pkg.Path))
	case "msi":
		err = pkg.RemoveMSI()
	default:
		for index := range len(pkg.Definition.Companions) + 1 {
			err = os.Remove(pkg.BinaryPath(index))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		err = nil
	}

	if err != nil {
		return err
	}

	log.Successf("removed %s in %s\n", pkg.Definition.Name, time.Since(removeStarted).Round(time.Millisecond))

	return nil
}

func (pkg *Package) Expand(value, version string) string {
	extension := ""

	if runtime.GOOS == "windows" {
		extension = ".exe"
	}

	replacer := strings.NewReplacer("{name}", pkg.Definition.Name, "{binary}", pkg.Definition.Binary, "{version}", version, "{tag}", pkg.Definition.Prefix+version, "{os}", runtime.GOOS, "{arch}", runtime.GOARCH, "{ext}", extension)

	return replacer.Replace(value)
}

func (pkg *Package) BinaryPath(index int) string {
	if index == 0 {
		return pkg.Path
	}

	name := pkg.Definition.Companions[index-1]

	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	return filepath.Join(filepath.Dir(pkg.Path), name)
}

func SelectPackages(names []string) ([]*Package, error) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if runtime.GOARCH != "amd64" {
		return nil, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	packages := make([]*Package, 0, len(packageCatalog))
	selected := make(map[string]struct{}, len(names))

	if len(names) == 0 {
		for index := range packageCatalog {
			definition := &packageCatalog[index]
			if _, supported := definition.Platforms[runtime.GOOS+"-"+runtime.GOARCH]; !supported {
				continue
			}

			pkg, err := ResolvePackage(definition)
			if err != nil {
				return nil, err
			}

			packages = append(packages, pkg)
		}

		return packages, nil
	}

	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if _, exists := selected[name]; exists {
			continue
		}

		var definition *catalog.Package

		for index := range packageCatalog {
			if packageCatalog[index].Name == name {
				definition = &packageCatalog[index]

				break
			}
		}

		if definition == nil {
			return nil, fmt.Errorf("unknown package %q; use tap find to browse the catalog", name)
		}

		pkg, err := ResolvePackage(definition)
		if err != nil {
			return nil, err
		}

		selected[name] = struct{}{}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func ResolvePackage(definition *catalog.Package) (*Package, error) {
	target, supported := definition.Platforms[runtime.GOOS+"-"+runtime.GOARCH]
	if !supported {
		return nil, fmt.Errorf("%s does not support %s/%s", definition.Name, runtime.GOOS, runtime.GOARCH)
	}

	home, err := UserHomeDir()
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Definition: definition,
		Target:     target,
	}

	switch definition.Source {
	case "go":
		pkg.Path = GetGoBinaryPath()
	case "pace":
		pkg.Path = filepath.Join(filepath.Dir(GetGoBinaryPath()), pkg.Expand("{binary}{ext}", ""))
	case "zig":
		pkg.Path = GetZigBinaryPath()
	default:
		path := pkg.Expand(target.Path, "")
		programFiles := os.Getenv("ProgramFiles")

		if strings.Contains(path, "{programfiles}") && !filepath.IsAbs(programFiles) {
			return nil, errors.New("programfiles must be absolute for msi installs")
		}

		replacer := strings.NewReplacer("{home}", home, "{bin}", filepath.Dir(GetLocalBinaryPath(definition.Binary)), "{programfiles}", programFiles)
		pkg.Path = filepath.Clean(filepath.FromSlash(replacer.Replace(path)))
	}

	if !filepath.IsAbs(pkg.Path) {
		return nil, fmt.Errorf("%s: could not resolve an absolute install path", definition.Name)
	}

	return pkg, nil
}
