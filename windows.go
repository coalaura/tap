//go:build windows

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

	return RunCommandOrError("msiexec.exe", "/i", path, "/qn", "/norestart")
}

func InstallZig(ver semver.SemVer) error {
	path, err := DownloadZigFile(ver)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	dstDir := filepath.Dir(GetZigBinaryPath())
	parent := filepath.Dir(dstDir)

	dir, err := os.MkdirTemp(parent, ".upgrader-zig-*")
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

func InstallUPX(ver semver.SemVer) error {
	home, err := UserHomeDir()
	if err != nil {
		return err
	}

	tag := "v" + ver.String()
	asset := fmt.Sprintf("upx-%s-win64.zip", ver.String())

	path := filepath.Join(home, ".bin", "upx.exe")

	return InstallSingleBinaryFromZip("upx/upx", tag, asset, "upx.exe", path, ver, []string{"--version"})
}

func InstallStarship(ver semver.SemVer) error {
	home, err := UserHomeDir()
	if err != nil {
		return err
	}

	tag := "v" + ver.String()
	asset := "starship-x86_64-pc-windows-msvc.zip"

	path := filepath.Join(home, ".bin", "starship.exe")

	return InstallSingleBinaryFromZip("starship/starship", tag, asset, "starship.exe", path, ver, []string{"--version"})
}

func InstallBun(ver semver.SemVer) error {
	tag := "bun-v" + ver.String()
	asset := "bun-windows-x64.zip"

	return InstallSingleBinaryFromZip("oven-sh/bun", tag, asset, "bun.exe", GetBunBinaryPath(), ver, []string{"--version"})
}

func InstallCoreutils(ver semver.SemVer) error {
	home, err := UserHomeDir()
	if err != nil {
		return err
	}

	tag := ver.String()
	asset := fmt.Sprintf("coreutils-%s-x86_64-pc-windows-msvc.zip", ver.String())

	path, err := DownloadGitHubAssetTemp("uutils/coreutils", tag, asset, ".zip")
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

	src := filepath.Join(dir, fmt.Sprintf("coreutils-%s-x86_64-pc-windows-msvc", ver.String()), "coreutils.exe")
	dst := filepath.Join(home, ".bin", "coreutils.exe")

	err = ValidateBinary(src, []string{"--version"}, ver)
	if err != nil {
		return err
	}

	return CopyFile(src, dst)
}

func DefaultGitHubAssetName(binary, _ string) string {
	return binary + "-windows-amd64.exe"
}

func VersionedGitHubAssetName(binary, version string) string {
	return binary + "_v" + version + "_windows_amd64.exe"
}
