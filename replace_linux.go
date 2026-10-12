//go:build linux

package main

import "os"

func ReplaceFile(src, dst string) error {
	return os.Rename(src, dst)
}
