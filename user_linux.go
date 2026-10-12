//go:build linux

package main

import (
	"os"
	"os/user"
	"strconv"
	"strings"
)

func ResolveUserHomeDir() (string, error) {
	name := strings.TrimSpace(os.Getenv("SUDO_USER"))

	if os.Geteuid() == 0 && name != "" && name != "root" {
		account, err := user.Lookup(name)
		if err != nil {
			return "", err
		}

		return account.HomeDir, nil
	}

	return os.UserHomeDir()
}

func ChownToInvokingUser(path string) error {
	name := strings.TrimSpace(os.Getenv("SUDO_USER"))

	if os.Geteuid() != 0 || name == "" || name == "root" {
		return nil
	}

	account, err := user.Lookup(name)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(account.Gid)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}
