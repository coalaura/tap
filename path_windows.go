//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func GetGoBinaryPath() string {
	return filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin", "go.exe")
}

func GetZigBinaryPath() string {
	home, err := UserHomeDir()
	if err != nil {
		return "zig.exe"
	}

	return filepath.Join(home, ".zig", "zig.exe")
}

func GetLocalBinaryPath(name string) string {
	home, err := UserHomeDir()
	if err != nil {
		return name + ".exe"
	}

	return filepath.Join(home, ".bin", name+".exe")
}

func GetBunBinaryPath() string {
	home, err := UserHomeDir()
	if err != nil {
		return "bun.exe"
	}

	return filepath.Join(home, ".bun", "bin", "bun.exe")
}
