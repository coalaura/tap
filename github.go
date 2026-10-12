package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	for index := range release.Assets {
		asset := &release.Assets[index]
		if asset.Name != name {
			continue
		}

		_, err := parseSHA256(asset.Digest)
		if err != nil {
			return nil, fmt.Errorf("asset %s: %w", name, err)
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
	uri := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repository, url.PathEscape(tag))

	release, err := fetchGitHubRelease(uri)
	if err != nil {
		return nil, err
	}

	asset, err := release.FindAsset(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repository, err)
	}

	return asset, nil
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
