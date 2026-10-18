//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func retryElevated(ctx context.Context, arguments []string, cause error) error {
	_, denied := errors.AsType[*writePermissionError](cause)
	if !denied || os.Geteuid() == 0 {
		return cause
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("%w; cannot elevate because sudo is unavailable; rerun this command as root", cause)
	}

	executable, err := os.Executable()
	if err != nil {
		return errors.Join(cause, err)
	}

	log.Warnf("%s; retrying with sudo for root permissions\n", cause)

	args := make([]string, 0, len(arguments)+3)
	args = append(args, "--user=root", "--", executable)
	args = append(args, arguments...)

	command := exec.CommandContext(ctx, sudo, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err = command.Run()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return describeCommandError("sudo retry", command, err, nil)
	}

	return nil
}
