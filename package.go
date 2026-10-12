package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/coalaura/semver"
	"github.com/coalaura/tap/internal/catalog"
)

type Package struct {
	Definition *catalog.Package
	Target     catalog.Target
	Path       string
}

func (pkg *Package) Get() error {
	log.Infof("Checking %s version...\n", pkg.Definition.Name)

	local, err := pkg.ResolveCurrentVersion()
	if err != nil {
		return err
	}

	remote, err := pkg.FetchLatestVersion()
	if err != nil {
		return err
	}

	if !local.Equal(semver.NewEmptySemVer()) && !remote.HigherThan(local) {
		log.Subf("Already up-to-date (installed %s, latest %s)\n", local, remote)

		return nil
	}

	log.Infof("Installing %s %s...\n", pkg.Definition.Name, remote)

	err = pkg.Install(remote)
	if err != nil {
		return err
	}

	err = ValidateBinary(pkg.Path, pkg.Definition.Args, remote)
	if err != nil {
		return err
	}

	log.Successf("Installed %s %s.\n", pkg.Definition.Name, remote)

	return nil
}

func (pkg *Package) IsInstalled() (bool, error) {
	info, err := os.Stat(pkg.Path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", pkg.Path)
	}

	return true, nil
}

func (pkg *Package) Remove() error {
	installed, err := pkg.IsInstalled()
	if err != nil {
		return err
	}

	if !installed {
		log.Subf("%s is not installed.\n", pkg.Definition.Name)

		return nil
	}

	switch pkg.Definition.Source {
	case "go":
		err = RemoveGo()
	case "zig":
		err = os.RemoveAll(filepath.Dir(pkg.Path))
	default:
		err = os.Remove(pkg.Path)
	}

	if err != nil {
		return err
	}

	log.Successf("Removed %s.\n", pkg.Definition.Name)

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
	case "zig":
		pkg.Path = GetZigBinaryPath()
	default:
		path := pkg.Expand(target.Path, "")
		replacer := strings.NewReplacer("{home}", home, "{bin}", filepath.Dir(GetLocalBinaryPath(definition.Binary)))
		pkg.Path = filepath.Clean(filepath.FromSlash(replacer.Replace(path)))
	}

	if !filepath.IsAbs(pkg.Path) {
		return nil, fmt.Errorf("%s: could not resolve an absolute install path", definition.Name)
	}

	return pkg, nil
}
