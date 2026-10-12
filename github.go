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

func (pkg *Package) FetchLatestVersion() (semver.SemVer, error) {
	switch pkg.Definition.Source {
	case "go":
		return FetchLatestGoVersion()
	case "zig":
		return FetchLatestZigVersion()
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", pkg.Definition.Repository)

	resp, err := githubRequest(endpoint)
	if err != nil {
		return semver.Invalid, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return semver.Invalid, err
	}

	var release GitHubRelease

	err = json.Unmarshal(body, &release)
	if err != nil {
		return semver.Invalid, err
	}

	if release.Draft || release.Prerelease {
		return semver.Invalid, errors.New("latest release is not stable")
	}

	return ParseVersionTag(release.TagName, pkg.Definition.Prefix)
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

	for _, asset := range release.Assets {
		if asset.Name != name {
			continue
		}

		_, err = parseSHA256(asset.Digest)
		if err != nil {
			return nil, fmt.Errorf("asset %s: %w", name, err)
		}

		return &asset, nil
	}

	return nil, fmt.Errorf("asset %s not found in %s release %s", name, repository, tag)
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
