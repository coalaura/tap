package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/coalaura/semver"
)

const tapRepository = "coalaura/tap"

func UpdateTap() error {
	if runtime.GOARCH != "amd64" || runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return fmt.Errorf("tap does not support %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	local := semver.Invalid

	if Version != "dev" {
		var err error

		local, err = semver.ParseSemVer(Version, true)
		if err != nil {
			return fmt.Errorf("invalid tap version %q: %w", Version, err)
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// The previous process may have kept its renamed executable locked until it exited.
		os.Remove(executable + ".tap-old")
	}

	log.Infof("Checking tap version...\n")

	release, err := FetchLatestGitHubRelease(tapRepository)
	if err != nil {
		return err
	}

	remote, err := ParseVersionTag(release.TagName, "v")
	if err != nil {
		return err
	}

	if !local.IsInvalid() && !remote.HigherThan(local) {
		log.Subf("Already up-to-date (installed %s, latest %s)\n", local, remote)

		return nil
	}

	name := "tap-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	asset, err := release.FindAsset(name)
	if err != nil {
		return err
	}

	log.Infof("Installing tap %s...\n", remote)

	source, err := DownloadTempFile(asset.URL, filepath.Ext(name), asset.Digest)
	if err != nil {
		return err
	}

	defer os.Remove(source)

	err = os.Chmod(source, 0755)
	if err != nil {
		return err
	}

	err = ValidateBinary(source, []string{"--version"}, remote)
	if err != nil {
		return err
	}

	info, err := os.Stat(executable)
	if err != nil {
		return err
	}

	// Stage beside the executable so replacement stays on the same filesystem.
	directory, err := os.MkdirTemp(filepath.Dir(executable), ".tap-update-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(directory)

	staged := filepath.Join(directory, filepath.Base(executable))

	err = CopyFileMode(source, staged, info.Mode().Perm())
	if err != nil {
		return err
	}

	err = ReplaceTapExecutable(staged, executable)
	if err != nil {
		return fmt.Errorf("replace tap executable: %w", err)
	}

	log.Successf("Updated tap to %s.\n", remote)

	return nil
}
