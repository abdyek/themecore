package themecore

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// writeSourceDir writes a development directory to a new temporary directory
// and returns it with the contract of testdata/package. change may edit the
// files before they are written; a nil value removes a file.
func writeSourceDir(t *testing.T, change func(files map[string][]byte)) (string, []byte) {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return b
	}
	files := map[string][]byte{
		"theme.toml":       read("testdata/source/theme.toml"),
		"attributes.toml":  read("testdata/package/attributes.toml"),
		"LICENSE":          read("testdata/package/LICENSE"),
		"assets/theme.css": FallbackSource(),
		"README.md":        []byte("not part of the package"),
		".dev/server":      []byte("not part of the package"),
	}
	if change != nil {
		change(files)
	}
	dir := t.TempDir()
	for name, b := range files {
		if b == nil {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets", "fonts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir, read("testdata/package/contract.toml")
}

func TestCheckSource(t *testing.T) {
	dir, contract := writeSourceDir(t, nil)
	p, report, err := CheckSource(dir, contract)
	if err != nil {
		t.Fatalf("CheckSource: %v", err)
	}
	var paths []string
	for path := range p.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if want := []string{"LICENSE", "assets/theme.css", "attributes.toml", "contract.toml"}; !reflect.DeepEqual(paths, want) {
		t.Errorf("package files = %v, want %v", paths, want)
	}
	if string(p.Files["assets/theme.css"]) != string(FallbackCSS()) {
		t.Errorf("assets/theme.css is not the canonical CSS")
	}
	if string(p.Files["contract.toml"]) != string(contract) {
		t.Errorf("contract.toml is not the given contract")
	}
	if report.Version != 1 {
		t.Errorf("Version = %d, want 1", report.Version)
	}
	if len(report.Evidence) == 0 || report.Evidence[0].Feature != "pkg.attributes" {
		t.Errorf("Evidence = %v, want it to start with pkg.attributes", report.Evidence)
	}
	if report.Warnings != nil {
		t.Errorf("Warnings = %v, want nil", report.Warnings)
	}
}

func TestCheckSourceRejects(t *testing.T) {
	cases := []struct {
		name   string
		change func(files map[string][]byte)
		want   string
	}{
		{"theme.toml missing", func(f map[string][]byte) { f["theme.toml"] = nil }, "theme.toml: missing"},
		{"dev/dev", func(f map[string][]byte) {
			f["theme.toml"] = []byte(strings.Replace(string(f["theme.toml"]), "acme/minimal", "dev/dev", 1))
		}, "dev/dev is reserved"},
		{"asset type", func(f map[string][]byte) { f["assets/notes.txt"] = []byte("x") }, "assets/notes.txt: not an allowed asset type"},
		{"asset path", func(f map[string][]byte) { f["assets/Bad.png"] = []byte("x") }, "assets/Bad.png"},
		{"theme.css missing", func(f map[string][]byte) { f["assets/theme.css"] = nil }, "package: assets/theme.css: missing"},
		{"theme.css grammar", func(f map[string][]byte) { f["assets/theme.css"] = []byte("html { color: red }") }, "theme.css"},
		{"URL to a missing file", func(f map[string][]byte) { f["assets/theme.css"] = []byte("a { background: url(missing.png) }") }, "missing.png"},
		{"invalid attributes", func(f map[string][]byte) {
			f["attributes.toml"] = []byte(strings.Replace(string(f["attributes.toml"]), "post POST_BODY IMAGE", "blog IMAGE", 1))
		}, "attributes.toml: "},
		{"LICENSE not UTF-8", func(f map[string][]byte) { f["LICENSE"] = []byte{0xff} }, "LICENSE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, contract := writeSourceDir(t, tc.change)
			_, _, err := CheckSource(dir, contract)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCheckSourceRejectsInvalidContract(t *testing.T) {
	dir, _ := writeSourceDir(t, nil)
	_, _, err := CheckSource(dir, []byte("x = 1"))
	if err == nil || !strings.Contains(err.Error(), "contract.toml") {
		t.Fatalf("error %v does not mention contract.toml", err)
	}
}

func TestCheckSourceRejectsSymlink(t *testing.T) {
	dir, contract := writeSourceDir(t, nil)
	if err := os.Symlink(filepath.Join(dir, "LICENSE"), filepath.Join(dir, "assets", "link.png")); err != nil {
		t.Skipf("symlink: %v", err)
	}
	_, _, err := CheckSource(dir, contract)
	if err == nil || !strings.Contains(err.Error(), "assets/link.png: not a regular file or directory") {
		t.Fatalf("error %v does not reject the symlink", err)
	}
}
