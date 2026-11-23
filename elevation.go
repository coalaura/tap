package main

import (
	"errors"
	"os"
)

type writePermissionError struct {
	cause error
}

func (err *writePermissionError) Error() string {
	return err.cause.Error()
}

func (err *writePermissionError) Unwrap() error {
	return err.cause
}

// Only destination filesystem failures qualify, never download or executable launch errors.
func writeError(err error) error {
	if !errors.Is(err, os.ErrPermission) {
		return err
	}

	return &writePermissionError{cause: err}
}
