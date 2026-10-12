//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coalaura/semver"
)

func InstallGo(ver semver.SemVer) error {
	path, err := DownloadGoFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("/usr/local", ".upgrader-go-*")
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

	dir, err := os.MkdirTemp("/usr/local", ".upgrader-zig-*")
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

func InstallUPX(ver semver.SemVer) error {
	tag := "v" + ver.String()
	asset := fmt.Sprintf("upx-%s-amd64_linux.tar.xz", ver.String())

	path, err := DownloadGitHubAssetTemp("upx/upx", tag, asset, ".tar.xz")
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("", "upgrader-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractTarXzFile(path, dir)
	if err != nil {
		return err
	}

	src, err := FindFile(dir, "upx")
	if err != nil {
		return err
	}

	err = ValidateBinary(src, []string{"--version"}, ver)
	if err != nil {
		return err
	}

	return CopyFileMode(src, "/usr/local/bin/upx", 0755)
}

func InstallStarship(ver semver.SemVer) error {
	tag := "v" + ver.String()
	asset := "starship-x86_64-unknown-linux-gnu.tar.gz"

	return InstallSingleBinaryFromTarGz("starship/starship", tag, asset, "starship", "/usr/local/bin/starship", ver, []string{"--version"})
}

func InstallBun(ver semver.SemVer) error {
	tag := "bun-v" + ver.String()
	asset := "bun-linux-x64.zip"

	path, err := DownloadGitHubAssetTemp("oven-sh/bun", tag, asset, ".zip")
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("", "upgrader-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractZipFile(path, dir)
	if err != nil {
		return err
	}

	src := filepath.Join(dir, "bun-linux-x64", "bun")
	dst := GetBunBinaryPath()

	binDir := filepath.Dir(dst)
	bunDir := filepath.Dir(binDir)

	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		return err
	}

	err = ChownToInvokingUser(bunDir)
	if err != nil {
		return err
	}

	err = ChownToInvokingUser(binDir)
	if err != nil {
		return err
	}

	err = os.Chmod(src, 0755)
	if err != nil {
		return err
	}

	err = ValidateBinary(src, []string{"--version"}, ver)
	if err != nil {
		return err
	}

	err = CopyFileMode(src, dst, 0755)
	if err != nil {
		return err
	}

	return ChownToInvokingUser(dst)
}

func InstallCoreutils(ver semver.SemVer) error {
	tag := ver.String()
	asset := fmt.Sprintf("coreutils-%s-x86_64-unknown-linux-gnu.tar.gz", ver.String())

	path, err := DownloadGitHubAssetTemp("uutils/coreutils", tag, asset, ".tar.gz")
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dir, err := os.MkdirTemp("", "upgrader-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	err = ExtractTarGzFile(path, dir)
	if err != nil {
		return err
	}

	src := filepath.Join(dir, fmt.Sprintf("coreutils-%s-x86_64-unknown-linux-gnu", ver.String()), "coreutils")

	err = ValidateBinary(src, []string{"--version"}, ver)
	if err != nil {
		return err
	}

	return CopyFileMode(src, "/usr/local/bin/coreutils", 0755)
}

func DefaultGitHubAssetName(binary, _ string) string {
	return binary + "-linux-amd64"
}

func VersionedGitHubAssetName(binary, version string) string {
	return binary + "_v" + version + "_linux_amd64"
}
