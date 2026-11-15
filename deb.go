package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ulikunitz/xz"
)

func ExtractDebFile(path, targetDir string) error {
	file, err := OpenFileForReading(path)
	if err != nil {
		return err
	}

	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	var magic [8]byte

	_, err = io.ReadFull(file, magic[:])
	if err != nil {
		return err
	}

	if string(magic[:]) != "!<arch>\n" {
		return errors.New("invalid deb archive signature")
	}

	remaining := info.Size() - int64(len(magic))

	// Debian uses fixed-width ar headers; reuse the buffer for each member.
	var header [60]byte

	for entry := range MaxArchiveEntries {
		if remaining == 0 {
			return errors.New("deb archive has no data tar member")
		}

		_, err = io.ReadFull(file, header[:])
		if err != nil {
			return err
		}

		remaining -= int64(len(header))

		if string(header[58:]) != "`\n" {
			return errors.New("invalid deb archive member header")
		}

		size, err := strconv.ParseInt(strings.TrimSpace(string(header[48:58])), 10, 64)
		if err != nil || size < 0 || size > MaxExtractedSize || size+size%2 > remaining {
			return errors.New("invalid deb archive member size")
		}

		name := strings.TrimSuffix(strings.TrimSpace(string(header[:16])), "/")

		if entry == 0 {
			if name != "debian-binary" || size != 4 {
				return errors.New("deb archive must start with debian-binary")
			}

			var version [4]byte

			_, err = io.ReadFull(file, version[:])
			if err != nil {
				return err
			}

			if string(version[:]) != "2.0\n" {
				return errors.New("unsupported deb archive version")
			}
		} else if name == "data.tar" || strings.HasPrefix(name, "data.tar.") {
			return extractDebData(io.LimitReader(file, size), name, targetDir)
		} else {
			_, err = file.Seek(size, io.SeekCurrent)
			if err != nil {
				return err
			}
		}

		if size%2 != 0 {
			var padding [1]byte

			_, err = io.ReadFull(file, padding[:])
			if err != nil {
				return err
			}

			if padding[0] != '\n' {
				return errors.New("invalid deb archive member padding")
			}
		}

		remaining -= size + size%2
	}

	return errors.New("deb archive has too many entries")
}

func extractDebData(reader io.Reader, name, targetDir string) error {
	switch name {
	case "data.tar":
		return ExtractTar(reader, targetDir)
	case "data.tar.gz":
		compressed, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}

		defer compressed.Close()

		return ExtractTar(compressed, targetDir)
	case "data.tar.xz":
		compressed, err := (xz.ReaderConfig{DictCap: MaxXZDictionary}).NewReader(reader)
		if err != nil {
			return err
		}

		return ExtractTar(compressed, targetDir)
	default:
		return fmt.Errorf("unsupported deb data compression: %s", name)
	}
}
