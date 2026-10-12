//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
)

func ReplaceTapExecutable(src, dst string) error {
	backup := dst + ".tap-old"

	// Reserve the backup name before moving the running executable out of the way.
	file, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		os.Remove(backup)

		return err
	}

	err = ReplaceFile(dst, backup)
	if err != nil {
		os.Remove(backup)

		return err
	}

	err = ReplaceFile(src, dst)
	if err != nil {
		rollbackErr := ReplaceFile(backup, dst)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("restore tap from %s: %w", backup, rollbackErr))
		}

		return err
	}

	// Windows may keep the old image locked; the next self-update removes it.
	os.Remove(backup)

	return nil
}
