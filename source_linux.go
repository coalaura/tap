//go:build linux

package main

func GoFileKind() string {
	return "archive"
}

func GoFileExtension() string {
	return ".tar.gz"
}

func ZigPlatform() string {
	return "x86_64-linux"
}

func ZigFileExtension() string {
	return ".tar.xz"
}
