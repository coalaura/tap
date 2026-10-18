package main

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func commandStartDetail(path string, cause error) string {
	switch {
	case errors.Is(cause, syscall.ENOENT):
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return ""
		}

		interpreter, kind := executableInterpreter(path)
		if interpreter != "" {
			_, err = os.Stat(interpreter)
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Sprintf("%s %q is missing (the executable itself exists)", kind, interpreter)
			}
		}

		return "executable exists, but a required loader or script interpreter may be missing: " + cause.Error()
	case errors.Is(cause, syscall.EACCES):
		return "permission denied; check executable and directory permissions and whether the filesystem is mounted noexec"
	case errors.Is(cause, syscall.ENOEXEC):
		return "unrecognized executable format; the file may target another OS/architecture, be corrupt, or be a script without a valid shebang"
	case errors.Is(cause, syscall.ELIBBAD):
		return "invalid ELF interpreter; the loader may be corrupt or incompatible"
	case errors.Is(cause, syscall.ETXTBSY):
		return "executable is open for writing; retry after the writer has closed it"
	default:
		return ""
	}
}

func commandExitDetail(failure *exec.ExitError) string {
	status, ok := failure.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() {
		detail := fmt.Sprintf("terminated by signal %d (%s)", status.Signal(), status.Signal())

		switch status.Signal() {
		case syscall.SIGILL:
			detail += " (illegal instruction; the binary may require unsupported CPU features)"
		case syscall.SIGSEGV:
			detail += " (invalid memory access)"
		}

		return detail
	}

	return fmt.Sprintf("exited with code %d", failure.ExitCode())
}

func executableInterpreter(path string) (string, string) {
	file, err := os.Open(path)
	if err != nil {
		return "", ""
	}

	defer file.Close()

	binary, err := elf.NewFile(file)
	if err == nil {
		for _, program := range binary.Progs {
			if program.Type != elf.PT_INTERP || program.Filesz < 2 || program.Filesz > 4096 {
				continue
			}

			// Bound reads from downloaded files, including malformed program headers.
			data := make([]byte, int(program.Filesz))
			_, readError := io.ReadFull(program.Open(), data)

			if readError != nil || data[len(data)-1] != 0 {
				return "", ""
			}

			end := bytes.IndexByte(data, 0)

			return string(data[:end]), "ELF interpreter"
		}

		return "", ""
	}

	var header [256]byte

	count, err := file.ReadAt(header[:], 0)
	if err != nil && !errors.Is(err, io.EOF) || count < 2 || header[0] != '#' || header[1] != '!' {
		return "", ""
	}

	line, _, newline := bytes.Cut(header[2:count], []byte{'\n'})
	line = bytes.TrimLeft(line, " \t")
	end := bytes.IndexAny(line, " \t")

	if end >= 0 {
		line = line[:end]
	} else if !newline && count == len(header) {
		return "", ""
	}

	// Linux retains carriage returns in shebang paths, so preserve them for diagnosis.
	return string(line), "script interpreter"
}
