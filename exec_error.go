package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/coalaura/plain/ansi"
)

type commandError struct {
	message string
	cause   error
}

func (failure *commandError) Error() string {
	return failure.message
}

func (failure *commandError) Unwrap() error {
	return failure.cause
}

func describeCommandError(label string, cmd *exec.Cmd, cause error, output []byte) error {
	var detail string

	exitError, exited := errors.AsType[*exec.ExitError](cause)
	if exited {
		detail = commandExitDetail(exitError)
	} else {
		detail = commandStartDetail(cmd.Path, cause)
		if detail == "" {
			switch {
			case errors.Is(cause, exec.ErrNotFound):
				detail = "executable not found in PATH"
			case errors.Is(cause, exec.ErrDot):
				detail = "refusing to run an executable found relative to the current directory; use an explicit path"
			case errors.Is(cause, os.ErrNotExist):
				detail = "executable or a required path does not exist"
			case errors.Is(cause, os.ErrPermission):
				detail = "permission denied when starting the executable"
			default:
				detail = cause.Error()
			}
		}

		detail = "could not start: " + detail
	}

	message := fmt.Sprintf("%s %q %s", label, cmd.String(), detail)
	output = bytes.TrimSpace(ansi.StripANSI(output))

	if len(output) > 0 {
		message += ": " + string(output)
	}

	return &commandError{message: message, cause: cause}
}
