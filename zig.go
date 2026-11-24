package main

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/coalaura/semver"
)

const (
	ZigMirrorListURL     = "https://ziglang.org/download/community-mirrors.txt"
	ZigMirrorListTimeout = 5 * time.Second
	ZigMirrorTimeout     = time.Minute
	ZigMirrorAttempts    = 2
	ZigMirrorMinimumRate = 1 << 20
)

func DownloadZigFile(ver semver.SemVer) (string, error) {
	file, err := FetchZigFile(ver)
	if err != nil {
		return "", err
	}

	mirrors, err := fetchZigMirrors()
	if err != nil {
		log.Subf("zig: mirror list unavailable: %v\n", err)
	}

	rand.Shuffle(len(mirrors), func(first, second int) {
		mirrors[first], mirrors[second] = mirrors[second], mirrors[first]
	})

	return downloadZigArchive(file, mirrors)
}

func fetchZigMirrors() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ZigMirrorListTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ZigMirrorListURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "tap/"+Version)

	resp, err := metadataClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ReadResponse(resp, MaxMetadataSize)
	if err != nil {
		return nil, err
	}

	return parseZigMirrors(string(body)), nil
}

func parseZigMirrors(body string) []string {
	mirrors := make([]string, 0, strings.Count(body, "\n")+1)

	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		mirror, err := url.Parse(line)
		if err != nil || mirror.Scheme != "https" || mirror.Hostname() == "" || mirror.User != nil || mirror.RawQuery != "" || mirror.Fragment != "" {
			continue
		}

		base := strings.TrimRight(mirror.String(), "/")

		if !slices.Contains(mirrors, base) {
			mirrors = append(mirrors, base)
		}
	}

	return mirrors
}

func downloadZigArchive(file *ZigFile, mirrors []string) (string, error) {
	origin, err := url.Parse(file.Tarball)
	if err != nil {
		return "", err
	}

	filename := path.Base(origin.Path)

	for _, mirror := range mirrors[:min(len(mirrors), ZigMirrorAttempts)] {
		log.Subf("zig: trying mirror %s\n", mirror)

		ctx, cancel := context.WithTimeout(context.Background(), ZigMirrorTimeout)
		archive, err := downloadTempFile(ctx, mirror+"/"+filename, ZigFileExtension(), file.SHA256, ZigMirrorMinimumRate)
		cancel()

		if err == nil {
			return archive, nil
		}

		log.Subf("zig: mirror %s failed: %v\n", mirror, err)
	}

	log.Subln("zig: downloading from ziglang.org")

	return DownloadTempFile(file.Tarball, ZigFileExtension(), file.SHA256)
}
