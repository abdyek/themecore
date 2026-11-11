package themecore

import (
	"archive/zip"
	"bytes"
	"hash/crc32"
	"io/fs"
	"strings"
	"testing"
)

type zipEntry struct {
	name   string
	data   []byte
	method uint16
	flags  uint16
	mode   fs.FileMode
}

func buildRawZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{
			Name:               e.name,
			Method:             e.method,
			Flags:              e.flags,
			CRC32:              crc32.ChecksumIEEE(e.data),
			CompressedSize64:   uint64(len(e.data)),
			UncompressedSize64: uint64(len(e.data)),
		}
		if e.mode != 0 {
			hdr.SetMode(e.mode)
		}
		fw, err := w.CreateRaw(hdr)
		if err != nil {
			t.Fatalf("buildRawZip: %v", err)
		}
		if _, err := fw.Write(e.data); err != nil {
			t.Fatalf("buildRawZip: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("buildRawZip: %v", err)
	}
	return buf.Bytes()
}

func TestReadZipRejectsPaths(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"license", "license: not allowed in the package root"},
		{"theme.toml", "theme.toml: not allowed in the package root"},
		{"assets", "assets: not allowed in the package root"},
		{"README.md", `README.md: segment "README.md" does not match [a-z0-9][a-z0-9._-]*`},
		{"fonts/a.woff2", "fonts/a.woff2: directories other than assets/ are not allowed"},
		{"assets/LICENSE", `assets/LICENSE: segment "LICENSE" does not match`},
		{"assets/.DS_Store", `segment ".DS_Store" does not match`},
		{"assets/Logo.png", `segment "Logo.png" does not match`},
		{"assets/../built.toml", `segment ".." does not match`},
		{"/built.toml", `segment "" does not match`},
		{"assets//a.png", `segment "" does not match`},
		{"assets" + string(rune(92)) + "a.png", "does not match"},
		{"assets/" + strings.Repeat("a", 65), "is longer than 64 bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			files := map[string][]byte{tc.path: []byte("x")}
			b := buildZip(t, files)
			_, err := readZip(t, b)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestReadZipRejectsTrees(t *testing.T) {
	cases := []struct {
		name  string
		files map[string][]byte
		dirs  []string
		want  string
	}{
		{
			name: "file and directory",
			files: map[string][]byte{
				"assets/a":       []byte("x"),
				"assets/a/b.png": []byte("x"),
			},
			want: "assets/a: both a file and a directory",
		},
		{
			name: "empty directory",
			files: map[string][]byte{
				"assets/theme.css": []byte("x"),
			},
			dirs: []string{"assets/", "assets/empty/"},
			want: "assets/empty/: empty directory",
		},
		{
			name: "directory outside assets",
			files: map[string][]byte{
				"assets/theme.css": []byte("x"),
			},
			dirs: []string{"x/"},
			want: "x/: directories other than assets/ are not allowed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := buildZip(t, tc.files, tc.dirs...)
			_, err := readZip(t, b)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestReadZipRejectsEntries(t *testing.T) {
	cases := []struct {
		name    string
		entries []zipEntry
		want    string
	}{
		{
			name: "duplicate",
			entries: []zipEntry{
				{name: "built.toml", data: []byte("a"), method: zip.Store},
				{name: "built.toml", data: []byte("b"), method: zip.Store},
			},
			want: "built.toml: duplicate entry",
		},
		{
			name: "encrypted",
			entries: []zipEntry{
				{name: "built.toml", data: []byte("x"), method: zip.Store, flags: 0x1},
			},
			want: "built.toml: encrypted entry",
		},
		{
			name: "compression method",
			entries: []zipEntry{
				{name: "built.toml", data: []byte("x"), method: 12},
			},
			want: "built.toml: unsupported compression method 12",
		},
		{
			name: "symlink",
			entries: []zipEntry{
				{name: "assets/theme.css", data: []byte("built.toml"), method: zip.Store, mode: fs.ModeSymlink | 0o777},
			},
			want: "assets/theme.css: not a regular file or directory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := buildRawZip(t, tc.entries...)
			_, err := readZip(t, b)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
