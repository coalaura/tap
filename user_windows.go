//go:build windows

package main

import "os"

func ResolveUserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func ChownToInvokingUser(path string) error {
	return nil
}
