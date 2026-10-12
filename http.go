package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	MaxDownloadSize = int64(2 << 30)
	MaxMetadataSize = int64(8 << 20)
)

var (
	downloadClient = &http.Client{
		Timeout: 15 * time.Minute,
	}

	metadataClient = &http.Client{
		Timeout: 30 * time.Second,
	}
)

func DownloadTempFile(url, ext, expectedHash string) (string, error) {
	file, err := os.CreateTemp("", "upgrader-*"+ext)
	if err != nil {
		return "", err
	}

	path := file.Name()

	err = downloadToFile(url, expectedHash, file)
	if err != nil {
		file.Close()

		os.Remove(path)

		return "", err
	}

	return path, nil
}

func downloadToFile(url, expectedHash string, file *os.File) error {
	hash, err := parseSHA256(expectedHash)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "env-upgrader/1.0")

	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	if resp.ContentLength > MaxDownloadSize {
		return fmt.Errorf("download is too large (%d bytes)", resp.ContentLength)
	}

	hasher := sha256.New()

	reader := io.LimitReader(resp.Body, MaxDownloadSize+1)

	n, copyErr := io.Copy(io.MultiWriter(file, hasher), reader)

	syncErr := file.Sync()
	closeErr := file.Close()

	if copyErr != nil {
		return copyErr
	}

	if n > MaxDownloadSize {
		return fmt.Errorf("download exceeds %d bytes", MaxDownloadSize)
	}

	if n < 128 {
		return fmt.Errorf("copied only %d bytes (too small)", n)
	}

	if syncErr != nil {
		return syncErr
	}

	if closeErr != nil {
		return closeErr
	}

	actual := hasher.Sum(nil)

	if subtle.ConstantTimeCompare(actual, hash) != 1 {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(actual))
	}

	return nil
}

func ReadResponse(resp *http.Response, limit int64) ([]byte, error) {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request returned %s", resp.Status)
	}

	reader := io.LimitReader(resp.Body, limit+1)

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}

	return body, nil
}

func parseSHA256(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sha256:")

	if value == "" {
		return nil, errors.New("missing sha256 digest")
	}

	hash, err := hex.DecodeString(value)
	if err != nil || len(hash) != sha256.Size {
		return nil, errors.New("invalid sha256 digest")
	}

	return hash, nil
}
