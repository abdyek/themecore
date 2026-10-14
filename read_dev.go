package themecore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// devSymlinks are the paths of the dev directory that may be symlinks into
// the theme project (Umono, İmtiyaz #2).
var devSymlinks = map[string]bool{"assets": true, "attributes.toml": true, "contract.toml": true}

// readDev reads the dev directory dir of Umono's dev mode. assets/,
// attributes.toml and contract.toml may be symlinks, which must resolve
// inside root, the theme project; any other symlink is rejected. The package
// limits and the allowlist of the root of dir are checked as ReadDir does,
// and an empty directory is skipped. A file under assets/ that breaks the
// path grammar or is not a regular file is left out of the package and
// returned in assetErrs.
func readDev(dir, root string) (p Package, assetErrs map[string]error, err error) {
	if !hardLinksDetected {
		return Package{}, nil, errors.New("package: hard links cannot be detected on this platform")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return Package{}, nil, err
	}
	if realRoot, err = filepath.Abs(realRoot); err != nil {
		return Package{}, nil, err
	}

	assetErrs = map[string]error{}
	var entries []entry
	paths := map[string]string{}
	add := func(rel, path string, info fs.FileInfo) {
		entries = append(entries, entry{path: rel, size: info.Size()})
		paths[rel] = path
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		return Package{}, nil, err
	}
	for _, item := range items {
		name := item.Name()
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if err != nil {
			return Package{}, nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if !devSymlinks[name] {
				return Package{}, nil, fmt.Errorf("package: %s: not a regular file or directory", name)
			}
			if path, err = filepath.EvalSymlinks(path); err != nil {
				return Package{}, nil, err
			}
			if !insideDir(realRoot, path) {
				return Package{}, nil, fmt.Errorf("package: %s: symlink points outside the theme project", name)
			}
			if info, err = os.Stat(path); err != nil {
				return Package{}, nil, err
			}
		}
		if info.IsDir() {
			if err := checkDirPath(name); err != nil {
				return Package{}, nil, err
			}
			if err := walkDevAssets(path, add, assetErrs); err != nil {
				return Package{}, nil, err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return Package{}, nil, fmt.Errorf("package: %s: not a regular file or directory", name)
		}
		if isHardLink(info) {
			return Package{}, nil, fmt.Errorf("package: %s: hard link", name)
		}
		add(name, path, info)
	}

	if err := checkEntries(entries); err != nil {
		return Package{}, nil, err
	}
	if err := checkContent(entries, nil); err != nil {
		return Package{}, nil, err
	}
	files := map[string][]byte{}
	var total int64
	for _, e := range entries {
		f, err := os.Open(paths[e.path])
		if err != nil {
			return Package{}, nil, err
		}
		data, err := readLimited(f, e.path, &total)
		f.Close()
		if err != nil {
			return Package{}, nil, err
		}
		files[e.path] = data
	}
	if len(assetErrs) == 0 {
		assetErrs = nil
	}
	return Package{Files: files}, assetErrs, nil
}

// walkDevAssets walks the assets directory at path. Every regular file with
// a valid path is passed to add; any other file is recorded in assetErrs.
func walkDevAssets(path string, add func(rel, path string, info fs.FileInfo), assetErrs map[string]error) error {
	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		sub, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		rel := "assets/" + filepath.ToSlash(sub)
		if !d.Type().IsRegular() {
			assetErrs[rel] = fmt.Errorf("package: %s: not a regular file or directory", rel)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if isHardLink(info) {
			assetErrs[rel] = fmt.Errorf("package: %s: hard link", rel)
			return nil
		}
		if err := checkFilePath(rel); err != nil {
			assetErrs[rel] = err
			return nil
		}
		add(rel, p, info)
		return nil
	})
}

// insideDir reports whether path is dir or inside it. Both are absolute and
// have no symlinks.
func insideDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
