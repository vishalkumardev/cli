package platform

import "runtime"

// Info returns the current platform details.
func Info() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}
