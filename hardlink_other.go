//go:build !unix

package themecore

import "io/fs"

// hardLinksDetected reports whether isHardLink can tell hard links apart on
// this platform. ReadDir fails where it cannot.
const hardLinksDetected = false

// isHardLink is never called on this platform: ReadDir fails before.
func isHardLink(info fs.FileInfo) bool {
	return false
}
