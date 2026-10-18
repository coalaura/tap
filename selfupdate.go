package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

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

	log.Infof("checking tap\n")

	checkStarted := time.Now()

	release, err := FetchLatestGitHubRelease(tapRepository)
	if err != nil {
		return err
	}

	remote, err := ParseVersionTag(release.TagName, "v")
	if err != nil {
		return err
	}

	if !local.IsInvalid() && !remote.HigherThan(local) {
		log.Subf("up to date: %s (checked in %s)\n", local, time.Since(checkStarted).Round(time.Millisecond))

		return nil
	}

	log.Subf("latest %s (checked in %s)\n", remote, time.Since(checkStarted).Round(time.Millisecond))

	name := "tap-" + runtime.GOOS + "-" + runtime.GOARCH

	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	asset, err := release.FindAsset(name)
	if err != nil {
		return err
	}

	err = checkDestinationPermissions(executable)
	if err != nil {
		return err
	}

	log.Stepf("updating tap %s\n", remote)

	installStarted := time.Now()

	source, err := DownloadTempFile(asset.URL, filepath.Ext(name), asset.Digest)
	if err != nil {
		return err
	}

	defer os.Remove(source)

	info, err := os.Stat(executable)
	if err != nil {
		return err
	}

	// Stage beside the executable for same-filesystem replacement and validation outside noexec temp directories.
	directory, err := os.MkdirTemp(filepath.Dir(executable), ".tap-update-*")
	if err != nil {
		return writeError(err)
	}

	defer os.RemoveAll(directory)

	staged := filepath.Join(directory, filepath.Base(executable))

	err = CopyFileMode(source, staged, info.Mode().Perm())
	if err != nil {
		return err
	}

	err = ValidateBinary(staged, []string{"--version"}, remote)
	if err != nil {
		return err
	}

	err = ReplaceTapExecutable(staged, executable)
	if err != nil {
		return fmt.Errorf("replace tap executable: %w", writeError(err))
	}

	log.Successf("updated tap %s in %s\n", remote, time.Since(installStarted).Round(time.Millisecond))

	return nil
}
