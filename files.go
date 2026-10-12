package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

func OpenFileForReading(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

func OpenFileForWriting(path string) (*os.File, error) {
	dir := filepath.Dir(path)

	_, err := os.Stat(dir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}

	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func CopyFile(src, dst string) error {
	return CopyFileMode(src, dst, 0)
}

func CopyFileMode(src, dst string, mode os.FileMode) error {
	in, err := OpenFileForReading(src)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dst)

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		in.Close()

		return err
	}

	out, err := os.CreateTemp(dir, ".upgrader-*")
	if err != nil {
		in.Close()

		return err
	}

	tmp := out.Name()

	defer os.Remove(tmp)

	_, copyErr := io.Copy(out, in)
	closeInErr := in.Close()

	syncErr := out.Sync()
	closeOutErr := out.Close()

	if copyErr != nil {
		return copyErr
	}

	if closeInErr != nil {
		return closeInErr
	}

	if syncErr != nil {
		return syncErr
	}

	if closeOutErr != nil {
		return closeOutErr
	}

	if mode == 0 {
		info, err := os.Stat(src)
		if err != nil {
			return err
		}

		mode = info.Mode().Perm()
	}

	err = os.Chmod(tmp, mode)
	if err != nil {
		return err
	}

	return ReplaceFile(tmp, dst)
}

func ReplaceDirectory(src, dst string) error {
	parent := filepath.Dir(dst)

	err := os.MkdirAll(parent, 0755)
	if err != nil {
		return err
	}

	backup, err := os.MkdirTemp(parent, ".upgrader-backup-*")
	if err != nil {
		return err
	}

	err = os.Remove(backup)
	if err != nil {
		return err
	}

	var hasBackup bool

	_, err = os.Stat(dst)
	if err == nil {
		err = os.Rename(dst, backup)
		if err != nil {
			return err
		}

		hasBackup = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	err = os.Rename(src, dst)
	if err != nil {
		if hasBackup {
			rollbackErr := os.Rename(backup, dst)
			if rollbackErr != nil {
				return errors.Join(err, rollbackErr)
			}
		}

		return err
	}

	if hasBackup {
		return os.RemoveAll(backup)
	}

	return nil
}
