package themecore

import (
	"fmt"
	"regexp"
	"strings"
)

// maxSegmentLen is the length limit of a path segment in bytes.
const maxSegmentLen = 64

// segmentPattern is the grammar of a path segment. It excludes hidden files,
// "..", empty segments, backslashes, upper case and URL escapes.
var segmentPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// rootFiles are the files allowed in the package root besides LICENSE.
var rootFiles = map[string]bool{
	"built.toml":      true,
	"contract.toml":   true,
	"checksums.json":  true,
	"attributes.toml": true,
}

// checkSegments checks each segment of path against the path grammar. shown is
// the path in messages.
func checkSegments(path, shown string) error {
	for _, segment := range strings.Split(path, "/") {
		if len(segment) > maxSegmentLen {
			return fmt.Errorf("package: %s: segment %q is longer than %d bytes", shown, segment, maxSegmentLen)
		}
		if !segmentPattern.MatchString(segment) {
			return fmt.Errorf("package: %s: segment %q does not match [a-z0-9][a-z0-9._-]*", shown, segment)
		}
	}
	return nil
}

// checkFilePath checks the path of a file against the path grammar and the
// allowlist of the package root. LICENSE is accepted as is; it is the only
// exception to the grammar.
func checkFilePath(path string) error {
	if path == "LICENSE" {
		return nil
	}
	if err := checkSegments(path, path); err != nil {
		return err
	}
	if !strings.Contains(path, "/") {
		if !rootFiles[path] {
			return fmt.Errorf("package: %s: not allowed in the package root", path)
		}
		return nil
	}
	if !strings.HasPrefix(path, "assets/") {
		return fmt.Errorf("package: %s: directories other than assets/ are not allowed", path)
	}
	return nil
}

// checkDirPath checks a directory, given without its trailing slash.
func checkDirPath(dir string) error {
	shown := dir + "/"
	if err := checkSegments(dir, shown); err != nil {
		return err
	}
	if dir != "assets" && !strings.HasPrefix(dir, "assets/") {
		return fmt.Errorf("package: %s: directories other than assets/ are not allowed", shown)
	}
	return nil
}

// checkContent checks the files and directories of a package against the
// package content allowlist: the path of each file, then each directory, then
// the tree.
func checkContent(entries []entry, dirs []string) error {
	for _, e := range entries {
		if err := checkFilePath(e.path); err != nil {
			return err
		}
	}
	for _, d := range dirs {
		if err := checkDirPath(d); err != nil {
			return err
		}
	}
	filePaths := make([]string, len(entries))
	for i, e := range entries {
		filePaths[i] = e.path
	}
	return checkTree(filePaths, dirs)
}

// checkTree checks the files and directories of a package together. dirs are
// the directories listed explicitly, without trailing slash.
func checkTree(files, dirs []string) error {
	set := make(map[string]struct{})
	for _, f := range files {
		set[f] = struct{}{}
	}
	for _, f := range files {
		prefix := f + "/"
		for other := range set {
			if strings.HasPrefix(other, prefix) {
				return fmt.Errorf("package: %s: both a file and a directory", f)
			}
		}
	}
	for _, d := range dirs {
		prefix := d + "/"
		found := false
		for f := range set {
			if strings.HasPrefix(f, prefix) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("package: %s: empty directory", prefix)
		}
	}
	return nil
}
