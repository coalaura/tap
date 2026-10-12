package main

import (
	"os"

	"github.com/coalaura/semver"
)

func InstallSingleBinaryFromTarGz(repository, tag, asset, binName, dstPath string, ver semver.SemVer, args []string) error {
	path, err := DownloadGitHubAssetTemp(repository, tag, asset, ".tar.gz")
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

	src, err := FindFile(dir, binName)
	if err != nil {
		return err
	}

	err = ValidateBinary(src, args, ver)
	if err != nil {
		return err
	}

	err = CopyFile(src, dstPath)
	if err != nil {
		return err
	}

	return os.Chmod(dstPath, 0755)
}

func InstallSingleBinaryFromZip(repository, tag, asset, binName, dstPath string, ver semver.SemVer, args []string) error {
	path, err := DownloadGitHubAssetTemp(repository, tag, asset, ".zip")
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

	src, err := FindFile(dir, binName)
	if err != nil {
		return err
	}

	err = ValidateBinary(src, args, ver)
	if err != nil {
		return err
	}

	return CopyFile(src, dstPath)
}
