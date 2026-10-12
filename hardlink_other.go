//go:build !unix

package themecore

import "io/fs"

// isHardLink is not detected on this platform.
func isHardLink(info fs.FileInfo) bool {
	return false
}
