package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	userHomeDir string
)

func UserHomeDir() (string, error) {
	if userHomeDir == "" {
		dir, err := ResolveUserHomeDir()
		if err != nil {
			return "", err
		}

		userHomeDir = dir
	}

	return userHomeDir, nil
}

func WarnMissingPath() {
	directory := filepath.Dir(GetLocalBinaryPath("tap"))
	if !filepath.IsAbs(directory) || IsInPath(directory) {
		return
	}

	log.Warnf("%s is not in path\n", directory)
}

func IsInPath(directory string) bool {
	directory = filepath.Clean(directory)

	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == "" {
			continue
		}

		path, err := filepath.Abs(entry)
		if err != nil {
			continue
		}

		if path == directory || runtime.GOOS == "windows" && strings.EqualFold(path, directory) {
			return true
		}
	}

	return false
}
