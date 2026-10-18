package main

import (
	"errors"
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

func commandStartDetail(_ string, cause error) string {
	switch {
	case errors.Is(cause, windows.ERROR_BAD_EXE_FORMAT):
		return "invalid executable format; the file may target another OS/architecture or be corrupt"
	case errors.Is(cause, windows.ERROR_EXE_MACHINE_TYPE_MISMATCH):
		return "executable architecture is incompatible with this system"
	case errors.Is(cause, windows.ERROR_DLL_NOT_FOUND), errors.Is(cause, windows.ERROR_MOD_NOT_FOUND):
		return "a required DLL or module could not be found"
	default:
		return ""
	}
}

func commandExitDetail(failure *exec.ExitError) string {
	status := windows.NTStatus(uint32(failure.ExitCode()))

	var detail string

	switch status {
	case windows.STATUS_DLL_NOT_FOUND:
		detail = "a required DLL could not be found"
	case windows.STATUS_INVALID_IMAGE_FORMAT:
		detail = "an executable or DLL has an invalid or incompatible format"
	case windows.STATUS_ILLEGAL_INSTRUCTION:
		detail = "illegal instruction; the binary may require unsupported CPU features"
	case windows.STATUS_ACCESS_VIOLATION:
		detail = "access violation (invalid memory access)"
	default:
		return fmt.Sprintf("exited with code %d", failure.ExitCode())
	}

	return fmt.Sprintf("exited with code 0x%08X (%s)", uint32(status), detail)
}
