package themecore

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	KB = 1 << 10
	MB = 1 << 20
)

func zeros(n int) []byte {
	return make([]byte, n)
}

func buildZip(t *testing.T, files map[string][]byte, dirs ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, d := range dirs {
		if _, err := w.Create(d); err != nil {
			t.Fatalf("buildZip: %v", err)
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("buildZip: %v", err)
		}
		if _, err := fw.Write(files[name]); err != nil {
			t.Fatalf("buildZip: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("buildZip: %v", err)
	}
	return buf.Bytes()
}

func readZip(t *testing.T, b []byte) (Package, error) {
	t.Helper()
	return ReadZip(bytes.NewReader(b), int64(len(b)))
}

func TestReadZipReadsFiles(t *testing.T) {
	files := map[string][]byte{
		"built.toml":       []byte("a"),
		"assets/theme.css": []byte("b"),
	}
	b := buildZip(t, files, "assets/")
	pkg, err := readZip(t, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string][]byte{
		"built.toml":       []byte("a"),
		"assets/theme.css": []byte("b"),
	}
	if !reflect.DeepEqual(pkg.Files, want) {
		t.Errorf("got %#v, want %#v", pkg.Files, want)
	}
}

func TestReadZipRejectsLargeBuiltFile(t *testing.T) {
	files := map[string][]byte{
		"built.toml": []byte("ok"),
	}
	b := buildZip(t, files)
	r := bytes.NewReader(b)
	_, err := ReadZip(r, 64*MB+1)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "built file is larger than 67108864 bytes") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadZipFileLimits(t *testing.T) {
	cases := []struct {
		path    string
		size    int
		wantErr string
	}{
		{"assets/theme.css", 2 * MB, ""},
		{"assets/theme.css", 2*MB + 1, "assets/theme.css: file is larger than 2097152 bytes"},
		{"built.toml", 1 * MB, ""},
		{"built.toml", 1*MB + 1, "built.toml: file is larger than 1048576 bytes"},
		{"contract.toml", 1*MB + 1, "contract.toml: file is larger than 1048576 bytes"},
		{"checksums.json", 1*MB + 1, "checksums.json: file is larger than 1048576 bytes"},
		{"attributes.toml", 1*MB + 1, "attributes.toml: file is larger than 1048576 bytes"},
		{"LICENSE", 256 * KB, ""},
		{"LICENSE", 256*KB + 1, "LICENSE: file is larger than 262144 bytes"},
		{"assets/fonts/a.woff2", 16 * MB, ""},
		{"assets/fonts/a.woff2", 16*MB + 1, "assets/fonts/a.woff2: file is larger than 16777216 bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			files := map[string][]byte{
				tc.path: zeros(tc.size),
			}
			b := buildZip(t, files)
			_, err := readZip(t, b)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestReadZipTotalSize(t *testing.T) {
	t.Run("at limit", func(t *testing.T) {
		files := map[string][]byte{
			"assets/a.png": zeros(16 * MB),
			"assets/b.png": zeros(16 * MB),
			"assets/c.png": zeros(16 * MB),
			"assets/d.png": zeros(16 * MB),
		}
		b := buildZip(t, files)
		if _, err := readZip(t, b); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("over limit", func(t *testing.T) {
		files := map[string][]byte{
			"assets/a.png":     zeros(16 * MB),
			"assets/b.png":     zeros(16 * MB),
			"assets/c.png":     zeros(16 * MB),
			"assets/d.png":     zeros(16 * MB),
			"assets/theme.css": []byte{0},
		}
		b := buildZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unpacked size is larger than 67108864 bytes") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestReadZipFileCount(t *testing.T) {
	t.Run("at limit", func(t *testing.T) {
		files := make(map[string][]byte)
		for i := 0; i < 1000; i++ {
			files[fmt.Sprintf("assets/f%d.png", i)] = []byte{0}
		}
		b := buildZip(t, files)
		if _, err := readZip(t, b); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("over limit", func(t *testing.T) {
		files := make(map[string][]byte)
		for i := 0; i < 1001; i++ {
			files[fmt.Sprintf("assets/f%d.png", i)] = []byte{0}
		}
		b := buildZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "more than 1000 files") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestReadZipDepth(t *testing.T) {
	t.Run("at limit", func(t *testing.T) {
		files := map[string][]byte{
			"assets/1/2/3/4/5/6/7/a.png": []byte{0},
		}
		b := buildZip(t, files)
		if _, err := readZip(t, b); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("over limit", func(t *testing.T) {
		files := map[string][]byte{
			"assets/1/2/3/4/5/6/7/8/a.png": []byte{0},
		}
		b := buildZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		wantErr := "assets/1/2/3/4/5/6/7/8/a.png: deeper than 8 directories"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error %q does not contain %q", err.Error(), wantErr)
		}
	})
}

func TestReadZipPathLength(t *testing.T) {
	p := "assets/" + strings.Repeat("a", 60) + "/" + strings.Repeat("b", 60) + "/" + strings.Repeat("c", 60) + "/" + strings.Repeat("d", 60) + "/"
	t.Run("at limit", func(t *testing.T) {
		path := p + "x.png"
		if len(path) != 256 {
			t.Fatalf("path length: got %d, want 256", len(path))
		}
		files := map[string][]byte{
			path: []byte{0},
		}
		b := buildZip(t, files)
		if _, err := readZip(t, b); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("over limit", func(t *testing.T) {
		path := p + "xy.png"
		if len(path) != 257 {
			t.Fatalf("path length: got %d, want 257", len(path))
		}
		files := map[string][]byte{
			path: []byte{0},
		}
		b := buildZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "path is longer than 256 bytes") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestReadZipRejectsNonZip(t *testing.T) {
	_, err := readZip(t, []byte("not a zip"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func buildLyingZip(t *testing.T, files map[string]int) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content := zeros(files[name])
		var cb bytes.Buffer
		fw, err := flate.NewWriter(&cb, flate.BestCompression)
		if err != nil {
			t.Fatalf("buildLyingZip: %v", err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("buildLyingZip: %v", err)
		}
		if err := fw.Close(); err != nil {
			t.Fatalf("buildLyingZip: %v", err)
		}
		compressed := cb.Bytes()
		hdr := &zip.FileHeader{
			Name:               name,
			Method:             zip.Deflate,
			CRC32:              crc32.ChecksumIEEE(content),
			CompressedSize64:   uint64(len(compressed)),
			UncompressedSize64: 1,
		}
		raw, err := w.CreateRaw(hdr)
		if err != nil {
			t.Fatalf("buildLyingZip: %v", err)
		}
		if _, err := raw.Write(compressed); err != nil {
			t.Fatalf("buildLyingZip: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("buildLyingZip: %v", err)
	}
	return buf.Bytes()
}

func writeFiles(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("writeFiles: %v", err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatalf("writeFiles: %v", err)
		}
	}
}

func TestReadZipCountsRealBytes(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		files := map[string]int{
			"assets/a.png": 16*MB + 1,
		}
		b := buildLyingZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		wantErr := "assets/a.png: entry is larger than its declared size"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error %q does not contain %q", err.Error(), wantErr)
		}
	})
	t.Run("total", func(t *testing.T) {
		files := map[string]int{
			"assets/a.png": 16 * MB,
			"assets/b.png": 16 * MB,
			"assets/c.png": 16 * MB,
			"assets/d.png": 16 * MB,
			"assets/e.png": 16 * MB,
		}
		b := buildLyingZip(t, files)
		_, err := readZip(t, b)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		wantErr := "assets/a.png: entry is larger than its declared size"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error %q does not contain %q", err.Error(), wantErr)
		}
	})
}

func TestReadDirReadsFiles(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string][]byte{
		"built.toml":       []byte("a"),
		"assets/theme.css": []byte("b"),
	})
	pkg, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string][]byte{
		"built.toml":       []byte("a"),
		"assets/theme.css": []byte("b"),
	}
	if !reflect.DeepEqual(pkg.Files, want) {
		t.Errorf("got %#v, want %#v", pkg.Files, want)
	}
}

func TestReadDirFileLimits(t *testing.T) {
	cases := []struct {
		path    string
		size    int
		wantErr string
	}{
		{"assets/theme.css", 2 * MB, ""},
		{"assets/theme.css", 2*MB + 1, "assets/theme.css: file is larger than 2097152 bytes"},
		{"assets/a.png", 16*MB + 1, "assets/a.png: file is larger than 16777216 bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string][]byte{
				tc.path: zeros(tc.size),
			})
			_, err := ReadDir(dir)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestReadDirDepth(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string][]byte{
		"assets/1/2/3/4/5/6/7/8/a.png": []byte{0},
	})
	_, err := ReadDir(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	wantErr := "assets/1/2/3/4/5/6/7/8/a.png: deeper than 8 directories"
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("error %q does not contain %q", err.Error(), wantErr)
	}
}

func TestReadDirRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string][]byte{
		"target.css": []byte("x"),
	})
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "target.css"), filepath.Join(dir, "assets", "theme.css")); err != nil {
		t.Skipf("symlink: %v", err)
	}
	_, err := ReadDir(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	wantErr := "assets/theme.css: not a regular file or directory"
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("error %q does not contain %q", err.Error(), wantErr)
	}
}
