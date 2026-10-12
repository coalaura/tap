package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

const (
	MaxArchiveEntries = 100000
	MaxExtractedSize  = int64(4 << 30)
	MaxXZDictionary   = 64 << 20
)

func ExtractTarGzFile(path, targetDir string) error {
	file, err := OpenFileForReading(path)
	if err != nil {
		return err
	}

	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	defer gzr.Close()

	return ExtractTar(gzr, targetDir)
}

func ExtractTarXzFile(path, targetDir string) error {
	file, err := OpenFileForReading(path)
	if err != nil {
		return err
	}

	defer file.Close()

	xzr, err := (xz.ReaderConfig{DictCap: MaxXZDictionary}).NewReader(file)
	if err != nil {
		return err
	}

	return ExtractTar(xzr, targetDir)
}

func ExtractTar(reader io.Reader, targetDir string) error {
	root, err := os.OpenRoot(targetDir)
	if err != nil {
		return err
	}

	defer root.Close()

	tr := tar.NewReader(reader)

	var (
		entries int
		total   int64
	)

	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		entries++

		if entries > MaxArchiveEntries {
			return errors.New("archive has too many entries")
		}

		name, err := filepath.Localize(path.Clean(header.Name))
		if err != nil {
			return errors.New("archive contains invalid path")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = root.MkdirAll(name, 0755)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			if header.Size < 0 || header.Size > MaxExtractedSize-total {
				return errors.New("archive exceeds extracted size limit")
			}

			err = root.MkdirAll(filepath.Dir(name), 0755)
			if err != nil {
				return err
			}

			out, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}

			n, copyErr := io.CopyN(out, tr, header.Size)

			var chmodErr error

			if copyErr == nil {
				chmodErr = out.Chmod(os.FileMode(header.Mode).Perm())
			}

			closeErr := out.Close()

			if copyErr != nil {
				return copyErr
			}

			if chmodErr != nil {
				return chmodErr
			}

			if closeErr != nil {
				return closeErr
			}

			total += n
		}
	}
}

func ExtractZipFile(archivePath, targetDir string) error {
	zrd, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}

	defer zrd.Close()

	if len(zrd.File) > MaxArchiveEntries {
		return errors.New("archive has too many entries")
	}

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(targetDir)
	if err != nil {
		return err
	}

	defer root.Close()

	var total int64

	for _, file := range zrd.File {
		name, err := filepath.Localize(path.Clean(file.Name))
		if err != nil {
			return errors.New("archive contains invalid path")
		}

		if file.FileInfo().IsDir() {
			err = root.MkdirAll(name, 0755)
			if err != nil {
				return err
			}

			continue
		}

		if !file.Mode().IsRegular() {
			return errors.New("archive contains unsupported file type")
		}

		if file.UncompressedSize64 > uint64(MaxExtractedSize-total) {
			return errors.New("archive exceeds extracted size limit")
		}

		err = root.MkdirAll(filepath.Dir(name), 0755)
		if err != nil {
			return err
		}

		in, err := file.Open()
		if err != nil {
			return err
		}

		out, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			in.Close()

			return err
		}

		remaining := MaxExtractedSize - total

		n, copyErr := io.Copy(out, io.LimitReader(in, remaining+1))

		closeOutErr := out.Close()
		closeInErr := in.Close()

		if copyErr != nil {
			return copyErr
		}

		if n > remaining {
			return errors.New("archive exceeds extracted size limit")
		}

		if closeOutErr != nil {
			return closeOutErr
		}

		if closeInErr != nil {
			return closeInErr
		}

		total += n
	}

	return nil
}

func ArchivePath(root, name string) (string, error) {
	name = filepath.FromSlash(name)

	if filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", errors.New("archive contains absolute path")
	}

	path := filepath.Join(root, filepath.Clean(name))

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("archive path escapes target directory")
	}

	return path, nil
}

func FindFile(root, name string) (string, error) {
	var found string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.EqualFold(info.Name(), name) {
			found = path

			return io.EOF
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, io.EOF) {
			return found, nil
		}

		return "", err
	}

	return "", os.ErrNotExist
}
