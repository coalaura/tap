package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	MaxDownloadSize = int64(2 << 30)
	MaxMetadataSize = int64(8 << 20)
)

type fallbackTransport struct {
	primary http.RoundTripper
	ipv4    http.RoundTripper
}

var (
	requestTransport = newHTTPTransport()

	downloadClient = &http.Client{
		Transport: requestTransport,
		Timeout:   15 * time.Minute,
	}

	metadataClient = &http.Client{
		Transport: requestTransport,
		Timeout:   30 * time.Second,
	}
)

func (transport *fallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := transport.primary.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusNotFound || req.Method != http.MethodGet && req.Method != http.MethodHead {
		return resp, nil
	}

	if req.Body != nil && req.Body != http.NoBody {
		return resp, nil
	}

	// Some download hosts return 404 over IPv6 even when the file exists over IPv4.
	resp.Body.Close()

	return transport.ipv4.RoundTrip(req)
}

func DownloadTempFile(url, ext, expectedHash string) (string, error) {
	file, err := os.CreateTemp("", "tap-*"+ext)
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

	req.Header.Set("User-Agent", "tap/"+Version)

	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", strings.ToLower(resp.Status))
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
		return nil, fmt.Errorf("request returned %s", strings.ToLower(resp.Status))
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

func newHTTPTransport() *fallbackTransport {
	ipv4 := http.DefaultTransport.(*http.Transport).Clone()
	dialContext := ipv4.DialContext

	ipv4.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialContext(ctx, "tcp4", address)
	}

	return &fallbackTransport{
		primary: http.DefaultTransport,
		ipv4:    ipv4,
	}
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
