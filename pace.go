package main

import (
	"debug/buildinfo"
	"fmt"

	"github.com/coalaura/semver"
)

func ResolveGoBuildVersion(path string) (semver.SemVer, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return semver.Invalid, err
	}

	return ParseVersionTag(info.GoVersion, "go")
}

func ValidatePaceGoVersion(version semver.SemVer) error {
	path := GetGoBinaryPath()

	installed, err := ResolveGoBuildVersion(path)
	if err != nil {
		return fmt.Errorf("pace %s requires Go %s at %s: %w", version, version, path, err)
	}

	if !installed.Equal(version) {
		return fmt.Errorf("pace %s requires Go %s at %s, got Go %s", version, version, path, installed)
	}

	return nil
}
