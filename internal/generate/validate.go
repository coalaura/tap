package main

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/coalaura/tap/internal/catalog"
)

func validatePackage(definition *catalog.Package) error {
	if !validName(definition.Name) || !validName(definition.Binary) {
		return errors.New("name and binary must contain only lowercase letters, digits, hyphens, underscores or dots, and start with a letter or digit")
	}

	if strings.TrimSpace(definition.Description) == "" {
		return errors.New("description is required")
	}

	switch definition.Source {
	case "github":
		owner, repository, ok := strings.Cut(definition.Repository, "/")
		if !ok || !validRepositoryPart(owner) || !validRepositoryPart(repository) {
			return errors.New("repository must be owner/name")
		}
	case "go", "zig":
		if definition.Binary != definition.Source {
			return errors.New("go and zig sources require the matching binary name")
		}
	default:
		return fmt.Errorf("unknown source %q (expected github, go or zig)", definition.Source)
	}

	if len(definition.Platforms) == 0 {
		return errors.New("at least one platform is required")
	}

	for platform, target := range definition.Platforms {
		if platform != "windows-amd64" && platform != "linux-amd64" {
			return fmt.Errorf("unsupported platform %q", platform)
		}

		err := validateTarget(definition, &target)
		if err != nil {
			return fmt.Errorf("%s: %w", platform, err)
		}

		definition.Platforms[platform] = target
	}

	return nil
}

func validateTarget(definition *catalog.Package, target *catalog.Target) error {
	if definition.Source != "github" {
		if target.Asset != "" || target.Executable != "" || target.Path != "" {
			return errors.New("go and zig use their official downloads and fixed install locations")
		}

		return nil
	}

	if target.Asset == "" {
		target.Asset = definition.Asset
	}

	if target.Path == "" {
		target.Path = "{bin}/{binary}{ext}"
	}

	err := validateTemplate(target.Asset, false)
	if err != nil {
		return fmt.Errorf("asset: %w", err)
	}

	if target.Asset == "" || strings.ContainsAny(target.Asset, "/\\") {
		return errors.New("asset must be a filename")
	}

	err = validateTemplate(target.Executable, false)
	if err != nil {
		return fmt.Errorf("executable: %w", err)
	}

	if target.Executable != "" && !validRelativePath(target.Executable) {
		return errors.New("executable must be a relative archive path using forward slashes")
	}

	err = validateTemplate(target.Path, true)
	if err != nil {
		return fmt.Errorf("path: %w", err)
	}

	if !strings.HasPrefix(target.Path, "{bin}/") && !strings.HasPrefix(target.Path, "{home}/") {
		return errors.New("path must start with {bin}/ or {home}/")
	}

	if !validRelativePath(target.Path) {
		return errors.New("path must use forward slashes without parent traversal")
	}

	return nil
}

func validateTemplate(value string, installPath bool) error {
	for value != "" {
		before, after, found := strings.Cut(value, "{")
		if strings.Contains(before, "}") {
			return errors.New("unmatched template brace")
		}

		if !found {
			return nil
		}

		variable, remaining, closed := strings.Cut(after, "}")
		if !closed {
			return errors.New("unmatched template brace")
		}

		switch variable {
		case "name", "binary", "os", "arch", "ext":
		case "version", "tag":
			if installPath {
				return errors.New("install paths cannot depend on a version")
			}
		case "bin", "home":
			if !installPath {
				return fmt.Errorf("{%s} is only valid in install paths", variable)
			}
		default:
			return fmt.Errorf("unknown template variable {%s}", variable)
		}

		value = remaining
	}

	return nil
}

func validName(value string) bool {
	if value == "" || value[0] == '-' || value[0] == '_' || value[0] == '.' {
		return false
	}

	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' || character == '_' || character == '.' {
			continue
		}

		return false
	}

	return true
}

func validRepositoryPart(value string) bool {
	return validName(strings.ToLower(value))
}

func validRelativePath(value string) bool {
	if strings.ContainsAny(value, "\\:\x00") || path.IsAbs(value) || path.Clean(value) != value || value == "." {
		return false
	}

	for component := range strings.SplitSeq(value, "/") {
		if component == ".." {
			return false
		}
	}

	return true
}
