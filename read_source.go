package themecore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// sourceRootFiles are the files of the root of a development directory that
// go into the package when they exist.
var sourceRootFiles = []string{"attributes.toml", "LICENSE"}

// readSource reads the development directory dir for build: theme.toml, and
// the package made of contract as contract.toml and of the files of dir that
// go into the package, assets/, attributes.toml and LICENSE. Other files of
// dir, such as .dev/ or dist/, are not read. The package limits and the
// content allowlist are checked as ReadDir does, except that an empty
// directory is skipped (themecore, Kararlar (Paket İçeriği), Karar 8).
func readSource(dir string, contract []byte) (Package, []byte, error) {
	if !hardLinksDetected {
		return Package{}, nil, errors.New("package: hard links cannot be detected on this platform")
	}
	manifest, err := readThemeTOML(dir)
	if err != nil {
		return Package{}, nil, err
	}

	entries := []entry{{path: "contract.toml", size: int64(len(contract))}}
	for _, name := range sourceRootFiles {
		info, err := os.Lstat(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Package{}, nil, err
		}
		if !info.Mode().IsRegular() {
			return Package{}, nil, fmt.Errorf("package: %s: not a regular file or directory", name)
		}
		if isHardLink(info) {
			return Package{}, nil, fmt.Errorf("package: %s: hard link", name)
		}
		entries = append(entries, entry{path: name, size: info.Size()})
	}

	assets := filepath.Join(dir, "assets")
	if _, err := os.Lstat(assets); err == nil {
		if err := filepath.WalkDir(assets, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("package: %s: not a regular file or directory", rel)
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if isHardLink(info) {
				return fmt.Errorf("package: %s: hard link", rel)
			}
			entries = append(entries, entry{path: rel, size: info.Size()})
			return nil
		}); err != nil {
			return Package{}, nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Package{}, nil, err
	}

	if err := checkEntries(entries); err != nil {
		return Package{}, nil, err
	}
	if err := checkContent(entries, nil); err != nil {
		return Package{}, nil, err
	}
	files := map[string][]byte{"contract.toml": contract}
	total := int64(len(contract))
	for _, e := range entries[1:] {
		f, err := os.Open(filepath.Join(dir, filepath.FromSlash(e.path)))
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
	return Package{Files: files}, manifest, nil
}

// readThemeTOML reads theme.toml of the development directory dir. It is
// not part of the package, but is held to the limit of the other manifests.
func readThemeTOML(dir string) ([]byte, error) {
	path := filepath.Join(dir, "theme.toml")
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("theme.toml: missing")
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("theme.toml: not a regular file")
	}
	if info.Size() > maxManifestSize {
		return nil, fmt.Errorf("theme.toml: file is larger than %d bytes", maxManifestSize)
	}
	return os.ReadFile(path)
}
