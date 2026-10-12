package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/coalaura/semver"
)

type GoRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}

type GoFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"`
}

type ZigFile struct {
	Tarball string `json:"tarball"`
	SHA256  string `json:"shasum"`
}

func FetchLatestGoVersion() (semver.SemVer, error) {
	resp, err := metadataClient.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return semver.Invalid, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return semver.Invalid, err
	}

	var releases []GoRelease

	err = json.Unmarshal(body, &releases)
	if err != nil {
		return semver.Invalid, err
	}

	latest := semver.Invalid

	for _, release := range releases {
		if !release.Stable {
			continue
		}

		version, err := ParseVersionTag(release.Version, "go")
		if err != nil {
			continue
		}

		if latest.IsInvalid() || version.HigherThan(latest) {
			latest = version
		}
	}

	if latest.IsInvalid() {
		return semver.Invalid, errors.New("no latest Go version found")
	}

	return latest, nil
}

func FetchLatestZigVersion() (semver.SemVer, error) {
	resp, err := metadataClient.Get("https://ziglang.org/download/index.json")
	if err != nil {
		return semver.Invalid, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return semver.Invalid, err
	}

	var versions map[string]json.RawMessage

	err = json.Unmarshal(body, &versions)
	if err != nil {
		return semver.Invalid, err
	}

	latest := semver.Invalid

	for name := range versions {
		version, err := semver.ParseSemVer(name, false)
		if err != nil {
			continue
		}

		if latest.IsInvalid() || version.HigherThan(latest) {
			latest = version
		}
	}

	if latest.IsInvalid() {
		return semver.Invalid, errors.New("no latest Zig version found")
	}

	return latest, nil
}

func FetchGoFile(ver semver.SemVer) (*GoFile, error) {
	resp, err := metadataClient.Get("https://go.dev/dl/?mode=json&include=all")
	if err != nil {
		return nil, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return nil, err
	}

	var releases []GoRelease

	err = json.Unmarshal(body, &releases)
	if err != nil {
		return nil, err
	}

	version := "go" + ver.String()

	for _, release := range releases {
		if release.Version != version {
			continue
		}

		for _, file := range release.Files {
			if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH && file.Kind == GoFileKind() {
				return &file, nil
			}
		}
	}

	return nil, fmt.Errorf("go %s download not found for %s/%s", ver, runtime.GOOS, runtime.GOARCH)
}

func FetchZigFile(ver semver.SemVer) (*ZigFile, error) {
	resp, err := metadataClient.Get("https://ziglang.org/download/index.json")
	if err != nil {
		return nil, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return nil, err
	}

	var versions map[string]map[string]json.RawMessage

	err = json.Unmarshal(body, &versions)
	if err != nil {
		return nil, err
	}

	version, ok := versions[ver.String()]
	if !ok {
		return nil, fmt.Errorf("zig %s download metadata not found", ver)
	}

	raw, ok := version[ZigPlatform()]
	if !ok {
		return nil, fmt.Errorf("zig %s download not found for %s/%s", ver, runtime.GOOS, runtime.GOARCH)
	}

	var file ZigFile

	err = json.Unmarshal(raw, &file)
	if err != nil {
		return nil, err
	}

	if file.Tarball == "" {
		return nil, errors.New("zig metadata is missing tarball URL")
	}

	_, err = parseSHA256(file.SHA256)
	if err != nil {
		return nil, fmt.Errorf("zig download: %w", err)
	}

	return &file, nil
}

func DownloadGoFile(ver semver.SemVer) (string, error) {
	file, err := FetchGoFile(ver)
	if err != nil {
		return "", err
	}

	return DownloadTempFile("https://go.dev/dl/"+file.Filename, GoFileExtension(), file.SHA256)
}

func DownloadZigFile(ver semver.SemVer) (string, error) {
	file, err := FetchZigFile(ver)
	if err != nil {
		return "", err
	}

	return DownloadTempFile(file.Tarball, ZigFileExtension(), file.SHA256)
}

func ParseVersionTag(tag, prefix string) (semver.SemVer, error) {
	if !strings.HasPrefix(tag, prefix) {
		return semver.Invalid, errors.New("tag prefix does not match")
	}

	return semver.ParseSemVer(strings.TrimPrefix(tag, prefix), false)
}
