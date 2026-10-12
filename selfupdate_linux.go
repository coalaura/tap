//go:build linux

package main

import (
	"os"
	"syscall"
)

func ReplaceTapExecutable(src, dst string) error {
	info, err := os.Stat(dst)
	if err != nil {
		return err
	}

	// Keep the installation's owner even when the update runs through sudo.
	stat := info.Sys().(*syscall.Stat_t)

	err = os.Chown(src, int(stat.Uid), int(stat.Gid))
	if err != nil {
		return err
	}

	return ReplaceFile(src, dst)
}
