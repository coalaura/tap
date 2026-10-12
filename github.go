package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/coalaura/semver"
)

type GitHubAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

type GitHubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []GitHubAsset `json:"assets"`
}

func (release *GitHubRelease) FindAsset(name string) (*GitHubAsset, error) {
	asset, err := release.findAsset(name)
	if err != nil {
		return nil, err
	}

	_, err = parseSHA256(asset.Digest)
	if err != nil {
		return nil, fmt.Errorf("asset %s: %w", name, err)
	}

	return asset, nil
}

func (release *GitHubRelease) findAsset(name string) (*GitHubAsset, error) {
	for index := range release.Assets {
		asset := &release.Assets[index]
		if asset.Name != name {
			continue
		}

		return asset, nil
	}

	return nil, fmt.Errorf("asset %s not found in release %s", name, release.TagName)
}

func (pkg *Package) FetchLatestVersion() (semver.SemVer, error) {
	switch pkg.Definition.Source {
	case "go":
		return FetchLatestGoVersion()
	case "zig":
		return FetchLatestZigVersion()
	}

	release, err := FetchLatestGitHubRelease(pkg.Definition.Repository)
	if err != nil {
		return semver.Invalid, err
	}

	return ParseVersionTag(release.TagName, pkg.Definition.Prefix)
}

func (pkg *Package) Download(version semver.SemVer) (string, error) {
	asset := pkg.Expand(pkg.Target.Asset, version.String())
	tag := pkg.Definition.Prefix + version.String()
	checksums := pkg.Expand(pkg.Definition.Checksums, version.String())

	info, err := fetchGitHubAsset(pkg.Definition.Repository, tag, asset, checksums)
	if err != nil {
		return "", err
	}

	return DownloadTempFile(info.URL, filepath.Ext(asset), info.Digest)
}

func FetchLatestGitHubRelease(repository string) (*GitHubRelease, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repository)

	release, err := fetchGitHubRelease(endpoint)
	if err != nil {
		return nil, err
	}

	if release.Draft || release.Prerelease {
		return nil, errors.New("latest release is not stable")
	}

	return release, nil
}

func DownloadGitHubAssetTemp(repository, tag, asset, ext string) (string, error) {
	info, err := FetchGitHubAsset(repository, tag, asset)
	if err != nil {
		return "", err
	}

	return DownloadTempFile(info.URL, ext, info.Digest)
}

func FetchGitHubAsset(repository, tag, name string) (*GitHubAsset, error) {
	return fetchGitHubAsset(repository, tag, name, "")
}

func fetchGitHubAsset(repository, tag, name, checksums string) (*GitHubAsset, error) {
	uri := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repository, url.PathEscape(tag))

	release, err := fetchGitHubRelease(uri)
	if err != nil {
		return nil, err
	}

	asset, err := release.findAsset(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repository, err)
	}

	if asset.Digest == "" && checksums != "" {
		checksumAsset, checksumErr := release.findAsset(checksums)
		if checksumErr != nil {
			return nil, checksumErr
		}

		asset.Digest, err = fetchGitHubChecksum(checksumAsset.URL, name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", checksums, err)
		}
	}

	_, err = parseSHA256(asset.Digest)
	if err != nil {
		return nil, fmt.Errorf("%s asset %s: %w", repository, name, err)
	}

	return asset, nil
}

func fetchGitHubChecksum(uri, name string) (string, error) {
	resp, err := metadataClient.Get(uri)
	if err != nil {
		return "", err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return "", err
	}

	return parseChecksumFile(body, name)
}

func fetchGitHubRelease(uri string) (*GitHubRelease, error) {
	resp, err := githubRequest(uri)
	if err != nil {
		return nil, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return nil, err
	}

	var release GitHubRelease

	err = json.Unmarshal(body, &release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

func githubRequest(uri string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "tap/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return metadataClient.Do(req)
}

func parseChecksumFile(body []byte, name string) (string, error) {
	digest := ""

	for line := range strings.SplitSeq(string(body), "\n") {
		var (
			fields [2]string
			count  int
		)

		for field := range strings.FieldsSeq(line) {
			if count == len(fields) {
				count++

				break
			}

			fields[count] = field
			count++
		}

		if count != len(fields) || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}

		if digest != "" {
			return "", fmt.Errorf("duplicate checksum for %s", name)
		}

		_, err := parseSHA256(fields[0])
		if err != nil {
			return "", fmt.Errorf("checksum for %s: %w", name, err)
		}

		digest = fields[0]
	}

	if digest == "" {
		return "", fmt.Errorf("checksum for %s not found", name)
	}

	return digest, nil
}
