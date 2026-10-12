package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/coalaura/plain/ansi"
	"github.com/coalaura/semver"
)

const (
	VersionCommandTimeout = 30 * time.Second
	InstallCommandTimeout = 15 * time.Minute
)

func (u *UpgradeConfig) ResolveCurrentVersion() (semver.SemVer, error) {
	path := u.Path

	if path == "" {
		var err error

		path, err = exec.LookPath(u.Binary)
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return semver.NewEmptySemVer(), nil
			}

			return semver.Invalid, err
		}
	} else {
		_, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return semver.NewEmptySemVer(), nil
			}

			return semver.Invalid, err
		}
	}

	return ResolveBinaryVersion(path, u.Args)
}

func ResolveBinaryVersion(path string, args []string) (semver.SemVer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), VersionCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return semver.Invalid, fmt.Errorf("version command timed out after %s", VersionCommandTimeout)
		}

		return semver.Invalid, err
	}

	out = ansi.StripANSI(out)

	if isDevVersionOutput(out) {
		return semver.NewEmptySemVer(), nil
	}

	versionText := findVersion(out)
	if len(versionText) == 0 {
		return semver.Invalid, errors.New("version not found")
	}

	version, err := semver.ParseSemVer(string(versionText), false)
	if err != nil {
		return semver.Invalid, err
	}

	return version, nil
}

func ValidateBinary(path string, args []string, expected semver.SemVer) error {
	version, err := ResolveBinaryVersion(path, args)
	if err != nil {
		return err
	}

	if !version.Equal(expected) {
		return fmt.Errorf("expected version %s, got %s", expected, version)
	}

	return nil
}

func RunCommandOrError(bin string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), InstallCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s timed out after %s", bin, InstallCommandTimeout)
		}

		out = bytes.TrimSpace(out)

		if len(out) == 0 {
			return err
		}

		return fmt.Errorf("%v: %s", err, string(out))
	}

	return nil
}

func isDevVersionOutput(output []byte) bool {
	output = bytes.TrimSpace(output)
	if bytes.ContainsAny(output, "\r\n") {
		return false
	}

	var (
		fields [3][]byte
		count  = 0
	)

	for field := range bytes.FieldsSeq(output) {
		if count == len(fields) {
			return false
		}

		fields[count] = field
		count++
	}

	if count == 0 || !bytes.Equal(fields[count-1], []byte("dev")) {
		return false
	}

	return count == 1 || bytes.Equal(fields[count-2], []byte("version"))
}

func findVersion(output []byte) []byte {
	for start := range len(output) {
		if !isVersionDigit(output[start]) || start > 0 && (isVersionDigit(output[start-1]) || output[start-1] == '.') {
			continue
		}

		end := start

		for end < len(output) && isVersionDigit(output[end]) {
			end++
		}

		if end == len(output) || output[end] != '.' {
			continue
		}

		end++
		minorStart := end

		for end < len(output) && isVersionDigit(output[end]) {
			end++
		}

		if end == minorStart {
			continue
		}

		if end < len(output) && output[end] == '.' {
			end++
			patchStart := end

			for end < len(output) && isVersionDigit(output[end]) {
				end++
			}

			if end == patchStart {
				continue
			}
		}

		if end < len(output) && output[end] == '.' {
			continue
		}

		return output[start:end]
	}

	return nil
}

func isVersionDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
