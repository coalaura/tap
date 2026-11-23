package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
)

type variantRecord struct {
	Variant string `yaml:"variant"`
	SHA256  string `yaml:"sha256"`
}

func (pkg *Package) InstalledVariant() (string, error) {
	if !pkg.hasDistinctVariants() {
		return pkg.Variant, nil
	}

	config, err := ReadConfig()
	if err != nil {
		return "", err
	}

	record := config.Packages[pkg.Definition.Name]

	if _, exists := pkg.Definition.Variants[record.Variant]; !exists {
		return "", nil
	}

	digest, err := binaryDigest(pkg.Path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	// A record only describes the exact binary Tap installed, not an external replacement.
	if digest != record.SHA256 {
		return "", nil
	}

	return record.Variant, nil
}

func (pkg *Package) RecordVariant() error {
	if !pkg.hasDistinctVariants() {
		return nil
	}

	config, err := ReadConfig()
	if err != nil {
		return err
	}

	digest, err := binaryDigest(pkg.Path)
	if err != nil {
		return err
	}

	if config.Packages == nil {
		config.Packages = make(map[string]variantRecord, 1)
	}

	config.Packages[pkg.Definition.Name] = variantRecord{
		Variant: pkg.Variant,
		SHA256:  digest,
	}

	return WriteConfig(config)
}

func (pkg *Package) RemoveVariantRecord() error {
	if len(pkg.Definition.Variants) == 0 {
		return nil
	}

	config, err := ReadConfig()
	if err != nil {
		return err
	}

	if _, exists := config.Packages[pkg.Definition.Name]; !exists {
		return nil
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	err = checkDestinationPermissions(path)
	if err != nil {
		return err
	}

	delete(config.Packages, pkg.Definition.Name)

	return WriteConfig(config)
}

func (pkg *Package) hasDistinctVariants() bool {
	if len(pkg.Definition.Variants) == 0 {
		return false
	}

	// Identical templates still select different binaries when they contain {variant}.
	if strings.Contains(pkg.Target.Asset, "{variant}") || strings.Contains(pkg.Target.Executable, "{variant}") || strings.Contains(pkg.Target.Path, "{variant}") {
		return true
	}

	platform := runtime.GOOS + "-" + runtime.GOARCH

	for _, variant := range pkg.Definition.Variants {
		target, supported := variant.Platforms[platform]
		if supported && target != pkg.Target {
			return true
		}
	}

	// Linux-only variants such as Rio's X11/Wayland share one Windows binary.
	return false
}

func binaryDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	hasher := sha256.New()

	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
