//go:build windows

package main

func (pkg *Package) checkPermissions(_ bool) error {
	return nil
}

func checkDestinationPermissions(_ string) error {
	return nil
}
