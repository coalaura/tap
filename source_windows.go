//go:build windows

package main

func GoFileKind() string {
	return "installer"
}

func GoFileExtension() string {
	return ".msi"
}

func ZigPlatform() string {
	return "x86_64-windows"
}

func ZigFileExtension() string {
	return ".zip"
}
