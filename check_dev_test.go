package themecore

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeDevDirs writes a theme project (root) and the dev directory of Umono
// (dir), whose contract.toml, attributes.toml and assets are symlinks into
// root, as the CLI creates them. change may edit the project files before
// they are written; a nil value removes a file.
func writeDevDirs(t *testing.T, change func(files map[string][]byte)) (dir, root string) {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return b
	}
	write := func(path string, b []byte) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	link := func(target, name string) {
		if err := os.Symlink(target, name); err != nil {
			t.Skipf("symlink: %v", err)
		}
	}
	files := map[string][]byte{
		".dev/contracts/example/blog@v1.0.toml": read("testdata/package/contract.toml"),
		"attributes.toml":                       read("testdata/package/attributes.toml"),
		"assets/theme.css":                      FallbackSource(),
	}
	if change != nil {
		change(files)
	}
	root = t.TempDir()
	for name, b := range files {
		if b != nil {
			write(filepath.Join(root, filepath.FromSlash(name)), b)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dir = filepath.Join(t.TempDir(), "themes", "dev", "dev", "0.0.0")
	write(filepath.Join(dir, "built.toml"), read("testdata/dev/built.toml"))
	link(filepath.Join(root, ".dev", "contracts", "example", "blog@v1.0.toml"), filepath.Join(dir, "contract.toml"))
	if files["attributes.toml"] != nil {
		link(filepath.Join(root, "attributes.toml"), filepath.Join(dir, "attributes.toml"))
	}
	link(filepath.Join(root, "assets"), filepath.Join(dir, "assets"))
	return dir, root
}

func TestCheckDev(t *testing.T) {
	dir, root := writeDevDirs(t, nil)
	r, err := CheckDev(dir, root)
	if err != nil {
		t.Fatalf("CheckDev: %v", err)
	}
	if r.Contract.ID != "example/blog" {
		t.Errorf("Contract.ID = %q, want %q", r.Contract.ID, "example/blog")
	}
	if r.Floor != 1 {
		t.Errorf("Floor = %d, want 1", r.Floor)
	}
	if len(r.Evidence) == 0 || r.Evidence[0].Feature != "pkg.attributes" {
		t.Errorf("Evidence = %v, want it to start with pkg.attributes", r.Evidence)
	}
	if r.CSSErr != nil || string(r.CSS) != string(FallbackCSS()) {
		t.Errorf("CSS = %q, CSSErr = %v, want the canonical fallback CSS", r.CSS, r.CSSErr)
	}
	if got, want := r.Attributes.Root("post"), map[string]string{"class": "layout-post"}; r.AttributesErr != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Attributes.Root(post) = %v, AttributesErr = %v, want %v", got, r.AttributesErr, want)
	}
	if len(r.AssetErrs) != 0 {
		t.Errorf("AssetErrs = %v, want none", r.AssetErrs)
	}
}

func TestCheckDevReturnsBuiltAndContractSource(t *testing.T) {
	dir, root := writeDevDirs(t, nil)
	r, err := CheckDev(dir, root)
	if err != nil {
		t.Fatalf("CheckDev: %v", err)
	}
	if r.Built.ID != "dev/dev" {
		t.Errorf("Built.ID = %q, want %q", r.Built.ID, "dev/dev")
	}
	if r.Built.Version != "0.0.0" {
		t.Errorf("Built.Version = %q, want %q", r.Built.Version, "0.0.0")
	}
	if r.Built.Contract != "example/blog@v1.0" {
		t.Errorf("Built.Contract = %q, want %q", r.Built.Contract, "example/blog@v1.0")
	}
	if r.Built.ThemeAPIVersion != 1 {
		t.Errorf("Built.ThemeAPIVersion = %d, want 1", r.Built.ThemeAPIVersion)
	}
	want, err := os.ReadFile("testdata/package/contract.toml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.ContractSource, want) {
		t.Errorf("ContractSource = %q, want the bytes of contract.toml", r.ContractSource)
	}
}

func TestCheckDevErrorsOfOneCheck(t *testing.T) {
	t.Run("theme.css grammar", func(t *testing.T) {
		dir, root := writeDevDirs(t, func(f map[string][]byte) { f["assets/theme.css"] = []byte("html { color: red }") })
		r, err := CheckDev(dir, root)
		if err != nil {
			t.Fatalf("CheckDev: %v", err)
		}
		if r.CSSErr == nil || r.CSS != nil {
			t.Errorf("CSS = %q, CSSErr = %v, want an error and no CSS", r.CSS, r.CSSErr)
		}
	})
	t.Run("theme.css missing", func(t *testing.T) {
		dir, root := writeDevDirs(t, func(f map[string][]byte) { f["assets/theme.css"] = nil })
		r, err := CheckDev(dir, root)
		if err != nil {
			t.Fatalf("CheckDev: %v", err)
		}
		if r.CSSErr == nil || !strings.Contains(r.CSSErr.Error(), "assets/theme.css: missing") {
			t.Errorf("CSSErr = %v, want assets/theme.css: missing", r.CSSErr)
		}
	})
	t.Run("invalid attributes", func(t *testing.T) {
		dir, root := writeDevDirs(t, func(f map[string][]byte) {
			f["attributes.toml"] = []byte(strings.Replace(string(f["attributes.toml"]), "post POST_BODY IMAGE", "blog IMAGE", 1))
		})
		r, err := CheckDev(dir, root)
		if err != nil {
			t.Fatalf("CheckDev: %v", err)
		}
		if r.AttributesErr == nil || r.Attributes.Root("post") != nil {
			t.Errorf("Attributes.Root(post) = %v, AttributesErr = %v, want an error and an empty table", r.Attributes.Root("post"), r.AttributesErr)
		}
	})
	t.Run("assets that break the rules", func(t *testing.T) {
		dir, root := writeDevDirs(t, func(f map[string][]byte) {
			f["assets/notes.txt"] = []byte("x")
			f["assets/Bad.png"] = []byte("x")
		})
		r, err := CheckDev(dir, root)
		if err != nil {
			t.Fatalf("CheckDev: %v", err)
		}
		for _, path := range []string{"assets/notes.txt", "assets/Bad.png"} {
			if r.AssetErrs[path] == nil {
				t.Errorf("AssetErrs[%s] = nil, want an error", path)
			}
		}
		if len(r.AssetErrs) != 2 {
			t.Errorf("AssetErrs = %v, want 2 entries", r.AssetErrs)
		}
	})
}

func TestCheckDevFatalErrors(t *testing.T) {
	t.Run("built.toml not dev/dev", func(t *testing.T) {
		dir, root := writeDevDirs(t, nil)
		b, _ := os.ReadFile(filepath.Join(dir, "built.toml"))
		if err := os.WriteFile(filepath.Join(dir, "built.toml"), []byte(strings.Replace(string(b), "dev/dev", "acme/minimal", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := CheckDev(dir, root)
		if err == nil || !strings.Contains(err.Error(), "built.toml: id: must be dev/dev") {
			t.Fatalf("error %v, want built.toml: id: must be dev/dev", err)
		}
	})
	t.Run("symlink outside root", func(t *testing.T) {
		dir, root := writeDevDirs(t, nil)
		outside := filepath.Join(t.TempDir(), "attributes.toml")
		b, _ := os.ReadFile("testdata/package/attributes.toml")
		if err := os.WriteFile(outside, b, 0o644); err != nil {
			t.Fatal(err)
		}
		os.Remove(filepath.Join(dir, "attributes.toml"))
		if err := os.Symlink(outside, filepath.Join(dir, "attributes.toml")); err != nil {
			t.Skipf("symlink: %v", err)
		}
		_, err := CheckDev(dir, root)
		if err == nil || !strings.Contains(err.Error(), "attributes.toml: symlink points outside the theme project") {
			t.Fatalf("error %v, want a symlink outside the theme project", err)
		}
	})
	t.Run("symlink not allowed", func(t *testing.T) {
		dir, root := writeDevDirs(t, nil)
		license := filepath.Join(root, "LICENSE")
		if err := os.WriteFile(license, []byte("MIT"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(license, filepath.Join(dir, "LICENSE")); err != nil {
			t.Skipf("symlink: %v", err)
		}
		_, err := CheckDev(dir, root)
		if err == nil || !strings.Contains(err.Error(), "LICENSE: not a regular file or directory") {
			t.Fatalf("error %v, want LICENSE: not a regular file or directory", err)
		}
	})
	t.Run("file not allowed at the root", func(t *testing.T) {
		dir, root := writeDevDirs(t, nil)
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := CheckDev(dir, root)
		if err == nil || !strings.Contains(err.Error(), "notes.txt") {
			t.Fatalf("error %v, want notes.txt rejected", err)
		}
	})
	t.Run("invalid contract", func(t *testing.T) {
		dir, root := writeDevDirs(t, func(f map[string][]byte) { f[".dev/contracts/example/blog@v1.0.toml"] = []byte("x = 1") })
		_, err := CheckDev(dir, root)
		if err == nil || !strings.Contains(err.Error(), "contract.toml") {
			t.Fatalf("error %v, want contract.toml rejected", err)
		}
	})
}

func TestParseBuiltStillRejectsDev(t *testing.T) {
	b, err := os.ReadFile("testdata/dev/built.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseBuilt(b); err == nil || !strings.Contains(err.Error(), "dev/dev is reserved") {
		t.Fatalf("ParseBuilt error %v, want dev/dev is reserved", err)
	}
}
