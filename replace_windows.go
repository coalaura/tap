//go:build windows

package main

import (
	"golang.org/x/sys/windows"
)

func ReplaceFile(src, dst string) error {
	srcPath, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}

	dstPath, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}

	return windows.MoveFileEx(srcPath, dstPath, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
