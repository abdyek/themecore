package themecore

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// fixturePackage returns the package of testdata/package with the fallback
// CSS as its canonical theme.css.
func fixturePackage(t *testing.T) Package {
	t.Helper()
	p := Package{Files: map[string][]byte{"assets/theme.css": FallbackCSS()}}
	for _, name := range []string{"built.toml", "contract.toml", "attributes.toml", "LICENSE", "checksums.json"} {
		b, err := os.ReadFile("testdata/package/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		p.Files[name] = b
	}
	return p
}

func TestCheckPackage(t *testing.T) {
	r, err := CheckPackage(fixturePackage(t))
	if err != nil {
		t.Fatalf("CheckPackage: %v", err)
	}
	if r.Contract.ID != "example/blog" {
		t.Errorf("Contract.ID = %q, want %q", r.Contract.ID, "example/blog")
	}
	if got, want := r.Attributes.Root("post"), map[string]string{"class": "layout-post"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Attributes.Root(post) = %v, want %v", got, want)
	}
	if r.Floor != 1 {
		t.Errorf("Floor = %d, want 1", r.Floor)
	}
	if len(r.Evidence) == 0 || r.Evidence[0] != (Signal{Feature: "pkg.attributes", IntroducedIn: 1, Where: "attributes.toml"}) {
		t.Errorf("Evidence = %v, want it to start with pkg.attributes", r.Evidence)
	}
}

func TestCheckPackageWithoutAttributes(t *testing.T) {
	p := fixturePackage(t)
	delete(p.Files, "attributes.toml")
	r, err := CheckPackage(p)
	if err != nil {
		t.Fatalf("CheckPackage: %v", err)
	}
	if got := r.Attributes.Root("post"); got != nil {
		t.Errorf("Attributes.Root(post) = %v, want nil", got)
	}
}

func TestCheckPackageRejects(t *testing.T) {
	missingURL, err := Transform([]byte("a { background: url(missing.png) }"))
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	cases := []struct {
		name   string
		change func(p Package)
		want   string
	}{
		{"built.toml missing", func(p Package) { delete(p.Files, "built.toml") }, "package: built.toml: missing"},
		{"contract.toml missing", func(p Package) { delete(p.Files, "contract.toml") }, "package: contract.toml: missing"},
		{"checksums.json missing", func(p Package) { delete(p.Files, "checksums.json") }, "package: checksums.json: missing"},
		{"theme.css missing", func(p Package) { delete(p.Files, "assets/theme.css") }, "package: assets/theme.css: missing"},
		{"contract mismatch", func(p Package) {
			p.Files["built.toml"] = []byte(strings.Replace(string(p.Files["built.toml"]), "example/blog@v1.0", "example/blog@v2.0", 1))
		}, "does not match contract.toml"},
		{"theme.css not canonical", func(p Package) { p.Files["assets/theme.css"] = []byte("a{color:red}") }, "theme.css"},
		{"URL to a missing file", func(p Package) { p.Files["assets/theme.css"] = missingURL }, "missing.png"},
		{"invalid attributes", func(p Package) {
			p.Files["attributes.toml"] = []byte(strings.Replace(string(p.Files["attributes.toml"]), "post POST_BODY IMAGE", "blog IMAGE", 1))
		}, "attributes.toml: "},
		{"LICENSE not UTF-8", func(p Package) { p.Files["LICENSE"] = []byte{0xff} }, "LICENSE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := fixturePackage(t)
			tc.change(p)
			_, err := CheckPackage(p)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
