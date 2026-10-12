package themecore

import (
	"fmt"
	"io"
	"strings"
)

// Package limits. A value equal to a limit is accepted.
const (
	maxBuiltSize    = 64 << 20  // the built file
	maxTotalSize    = 64 << 20  // all files unpacked
	maxAssetSize    = 16 << 20  // a file under assets/
	maxThemeCSSSize = 2 << 20   // assets/theme.css
	maxManifestSize = 1 << 20   // built.toml, checksums.json, contract.toml and attributes.toml, each
	maxLicenseSize  = 256 << 10 // LICENSE
	maxFiles        = 1000
	maxDepth        = 8   // directories in a path, assets included
	maxPathLen      = 256 // bytes of a path
)

// fileLimit returns the size limit of the file at path.
func fileLimit(path string) int64 {
	switch path {
	case "assets/theme.css":
		return maxThemeCSSSize
	case "built.toml", "checksums.json", "contract.toml", "attributes.toml":
		return maxManifestSize
	case "LICENSE":
		return maxLicenseSize
	default:
		return maxAssetSize
	}
}

// entry is a file of a package with the size its metadata declares.
type entry struct {
	path string
	size int64
}

// checkEntries checks the limits that the metadata alone shows. It runs before any file is read.
func checkEntries(entries []entry) error {
	if len(entries) > maxFiles {
		return fmt.Errorf("package: more than %d files", maxFiles)
	}
	var sum int64
	for _, e := range entries {
		if len(e.path) > maxPathLen {
			return fmt.Errorf("package: %s: path is longer than %d bytes", e.path, maxPathLen)
		}
		if strings.Count(e.path, "/") > maxDepth {
			return fmt.Errorf("package: %s: deeper than %d directories", e.path, maxDepth)
		}
		limit := fileLimit(e.path)
		if e.size > limit {
			return fmt.Errorf("package: %s: file is larger than %d bytes", e.path, limit)
		}
		sum += e.size
	}
	if sum > maxTotalSize {
		return fmt.Errorf("package: unpacked size is larger than %d bytes", maxTotalSize)
	}
	return nil
}

// readLimited reads from r counting its real bytes. It stops reading as soon as
// a limit is exceeded, so a file that grew after its metadata was read, or an
// entry with a wrong declared size, cannot exceed a limit. *total is the
// number of bytes already read from the package and is increased by the
// file's length.
func readLimited(r io.Reader, path string, total *int64) ([]byte, error) {
	limit := fileLimit(path)
	remaining := maxTotalSize - *total
	if remaining < limit {
		limit = remaining
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	limit = fileLimit(path)
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("package: %s: file is larger than %d bytes", path, limit)
	}
	if *total+int64(len(data)) > maxTotalSize {
		return nil, fmt.Errorf("package: unpacked size is larger than %d bytes", maxTotalSize)
	}
	*total += int64(len(data))
	return data, nil
}
