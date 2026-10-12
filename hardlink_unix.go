//go:build unix

package themecore

import (
	"io/fs"
	"syscall"
)

// isHardLink reports whether the file has more than one name.
func isHardLink(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Nlink > 1
}
