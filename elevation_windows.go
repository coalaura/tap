//go:build windows

package main

import "context"

func retryElevated(_ context.Context, _ []string, cause error) error {
	return cause
}
