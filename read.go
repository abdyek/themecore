package themecore

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// ReadZip reads a built file. size is the file's size from the file system's
// metadata. The package limits are checked before anything is read: the built
// file's size first, then the sizes the entries declare. While an entry is
// decompressed its real bytes are counted, and reading stops as soon as a
// limit is exceeded. Entry names, types, encryption and compression methods
// are checked before any entry is decompressed.
func ReadZip(r io.ReaderAt, size int64) (Package, error) {
	if size > maxBuiltSize {
		return Package{}, fmt.Errorf("package: built file is larger than %d bytes", maxBuiltSize)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Package{}, fmt.Errorf("package: not a zip file: %w", err)
	}
	seen := make(map[string]struct{})
	var entries []entry
	var dirs []string
	for _, f := range zr.File {
		if _, dup := seen[f.Name]; dup {
			return Package{}, fmt.Errorf("package: %s: duplicate entry", f.Name)
		}
		seen[f.Name] = struct{}{}
		if f.Flags&0x1 != 0 {
			return Package{}, fmt.Errorf("package: %s: encrypted entry", f.Name)
		}
		if f.Method != zip.Store && f.Method != zip.Deflate {
			return Package{}, fmt.Errorf("package: %s: unsupported compression method %d", f.Name, f.Method)
		}
		if strings.HasSuffix(f.Name, "/") {
			if !f.Mode().IsDir() {
				return Package{}, fmt.Errorf("package: %s: not a regular file or directory", f.Name)
			}
			dirs = append(dirs, strings.TrimSuffix(f.Name, "/"))
			continue
		}
		if !f.Mode().IsRegular() {
			return Package{}, fmt.Errorf("package: %s: not a regular file or directory", f.Name)
		}
		declared := int64(f.UncompressedSize64)
		if f.UncompressedSize64 > math.MaxInt64 {
			declared = math.MaxInt64
		}
		entries = append(entries, entry{path: f.Name, size: declared})
	}
	if err := checkEntries(entries); err != nil {
		return Package{}, err
	}
	if err := checkContent(entries, dirs); err != nil {
		return Package{}, err
	}
	files := make(map[string][]byte)
	var total int64
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return Package{}, err
		}
		data, err := readLimited(rc, f.Name, &total)
		rc.Close()
		if err != nil {
			if errors.Is(err, zip.ErrFormat) {
				return Package{}, fmt.Errorf("package: %s: entry is larger than its declared size", f.Name)
			}
			return Package{}, err
		}
		files[f.Name] = data
	}
	return Package{Files: files}, nil
}

// ReadDir reads the installed form of a package from dir. The package limits
// are checked with the sizes from the file system's metadata before any file is
// read. Symlinks are not followed: anything that is not a regular file or a
// directory is rejected, and so are hard links. Hard links can only be
// detected on Unix, so ReadDir fails on other platforms.
func ReadDir(dir string) (Package, error) {
	if !hardLinksDetected {
		return Package{}, errors.New("package: hard links cannot be detected on this platform")
	}
	var entries []entry
	var dirs []string
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			dirs = append(dirs, rel)
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
		return Package{}, err
	}
	if err := checkEntries(entries); err != nil {
		return Package{}, err
	}
	if err := checkContent(entries, dirs); err != nil {
		return Package{}, err
	}
	files := make(map[string][]byte)
	var total int64
	for _, e := range entries {
		f, err := os.Open(filepath.Join(dir, filepath.FromSlash(e.path)))
		if err != nil {
			return Package{}, err
		}
		data, err := readLimited(f, e.path, &total)
		f.Close()
		if err != nil {
			return Package{}, err
		}
		files[e.path] = data
	}
	return Package{Files: files}, nil
}
