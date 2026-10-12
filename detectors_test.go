package themecore

import (
	"strings"
	"testing"
)

func feature(t *testing.T, id string) Feature {
	t.Helper()
	for _, f := range Features {
		if f.ID == id {
			return f
		}
	}
	t.Fatalf("feature %q not found", id)
	return Feature{}
}

const attrsDoc = `schema = "https://umono.dev/schemas/theme-attributes/v1"

[address."post"]
class = "layout-post"

[address."home"]
class = "layout-home"

[address."post IMAGE"]
sizes = "100vw"

[address."post MAIN_MENU LINK"]
class = "main-menu-link"
`

func TestLedgerPackageRows(t *testing.T) {
	want := []string{
		"pkg.attributes",
		"attr.class",
		"attr.sizes",
		"attr.address.layout",
		"attr.address.layout-builtin",
		"attr.address.layout-component-builtin",
		"pkg.license",
		"assets.woff2",
		"assets.woff",
		"assets.png",
		"assets.jpeg",
		"assets.jpg",
		"assets.webp",
		"assets.avif",
		"assets.gif",
	}
	if len(Features) < len(want) {
		t.Fatalf("expected at least %d features, got %d", len(want), len(Features))
	}
	for i, id := range want {
		if Features[i].ID != id {
			t.Errorf("Features[%d].ID = %q, want %q", i, Features[i].ID, id)
		}
		if Features[i].IntroducedIn != 1 {
			t.Errorf("Features[%d] %q IntroducedIn = %d, want 1", i, id, Features[i].IntroducedIn)
		}
	}
}

func TestDetectors(t *testing.T) {
	cases := []struct {
		name  string
		id    string
		files map[string][]byte
		where string
		found bool
	}{
		{
			name:  "attributes present",
			id:    "pkg.attributes",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "attributes.toml",
			found: true,
		},
		{
			name:  "attributes absent",
			id:    "pkg.attributes",
			files: map[string][]byte{"built.toml": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "attributes unparsable",
			id:    "pkg.attributes",
			files: map[string][]byte{"attributes.toml": []byte("x = ")},
			where: "attributes.toml",
			found: true,
		},
		{
			name:  "class",
			id:    "attr.class",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "home",
			found: true,
		},
		{
			name:  "sizes",
			id:    "attr.sizes",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "post IMAGE",
			found: true,
		},
		{
			name:  "layout address",
			id:    "attr.address.layout",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "home",
			found: true,
		},
		{
			name:  "layout builtin address",
			id:    "attr.address.layout-builtin",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "post IMAGE",
			found: true,
		},
		{
			name:  "layout component builtin address",
			id:    "attr.address.layout-component-builtin",
			files: map[string][]byte{"attributes.toml": []byte(attrsDoc)},
			where: "post MAIN_MENU LINK",
			found: true,
		},
		{
			name:  "no layout address",
			id:    "attr.address.layout",
			files: map[string][]byte{"attributes.toml": []byte("schema = \"https://umono.dev/schemas/theme-attributes/v1\"\n\n[address.\"post IMAGE\"]\nclass = \"cover\"\n")},
			where: "",
			found: false,
		},
		{
			name:  "class unparsable",
			id:    "attr.class",
			files: map[string][]byte{"attributes.toml": []byte("x = ")},
			where: "",
			found: false,
		},
		{
			name:  "class without attributes",
			id:    "attr.class",
			files: map[string][]byte{"built.toml": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "license",
			id:    "pkg.license",
			files: map[string][]byte{"LICENSE": []byte("MIT")},
			where: "LICENSE",
			found: true,
		},
		{
			name:  "no license",
			id:    "pkg.license",
			files: map[string][]byte{"built.toml": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "woff2",
			id:    "assets.woff2",
			files: map[string][]byte{"assets/fonts/b.woff2": []byte("x"), "assets/fonts/a.woff2": []byte("x")},
			where: "assets/fonts/a.woff2",
			found: true,
		},
		{
			name:  "woff is not woff2",
			id:    "assets.woff",
			files: map[string][]byte{"assets/fonts/a.woff2": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "woff",
			id:    "assets.woff",
			files: map[string][]byte{"assets/a.woff": []byte("x")},
			where: "assets/a.woff",
			found: true,
		},
		{
			name:  "png",
			id:    "assets.png",
			files: map[string][]byte{"assets/a.png": []byte("x")},
			where: "assets/a.png",
			found: true,
		},
		{
			name:  "jpeg is not jpg",
			id:    "assets.jpg",
			files: map[string][]byte{"assets/a.jpeg": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "jpeg",
			id:    "assets.jpeg",
			files: map[string][]byte{"assets/a.jpeg": []byte("x")},
			where: "assets/a.jpeg",
			found: true,
		},
		{
			name:  "jpg",
			id:    "assets.jpg",
			files: map[string][]byte{"assets/a.jpg": []byte("x")},
			where: "assets/a.jpg",
			found: true,
		},
		{
			name:  "webp",
			id:    "assets.webp",
			files: map[string][]byte{"assets/a.webp": []byte("x")},
			where: "assets/a.webp",
			found: true,
		},
		{
			name:  "avif",
			id:    "assets.avif",
			files: map[string][]byte{"assets/a.avif": []byte("x")},
			where: "assets/a.avif",
			found: true,
		},
		{
			name:  "gif",
			id:    "assets.gif",
			files: map[string][]byte{"assets/a.gif": []byte("x")},
			where: "assets/a.gif",
			found: true,
		},
		{
			name:  "gif outside assets",
			id:    "assets.gif",
			files: map[string][]byte{"a.gif": []byte("x")},
			where: "",
			found: false,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := feature(t, c.id)
			where, found := f.Detect(Package{Files: c.files})
			if where != c.where || found != c.found {
				t.Errorf("Detect(%s) = (%q, %v), want (%q, %v)", c.id, where, found, c.where, c.found)
			}
		})
	}
}

const canonicalCSS = "@font-face{font-family:x}\n@layer a,b;\n@media print{@supports (display:grid){@scope (umono-root){a{color:red}}}}\n@keyframes k{from{opacity:0}}\n"

func TestLedgerCSSRows(t *testing.T) {
	want := []string{
		"css.at.media",
		"css.at.supports",
		"css.at.layer",
		"css.at.font-face",
		"css.at.keyframes",
	}
	if len(Features) < 20 {
		t.Fatalf("expected at least 20 features, got %d", len(Features))
	}
	for i, id := range want {
		j := 15 + i
		if Features[j].ID != id {
			t.Errorf("Features[%d].ID = %q, want %q", j, Features[j].ID, id)
		}
		if Features[j].IntroducedIn != 1 {
			t.Errorf("Features[%d] %q IntroducedIn = %d, want 1", j, id, Features[j].IntroducedIn)
		}
	}
}

func TestCSSDetectors(t *testing.T) {
	cases := []struct {
		name  string
		id    string
		files map[string][]byte
		where string
		found bool
	}{
		{
			name:  "media",
			id:    "css.at.media",
			files: map[string][]byte{"assets/theme.css": []byte(canonicalCSS)},
			where: "assets/theme.css:3",
			found: true,
		},
		{
			name:  "supports",
			id:    "css.at.supports",
			files: map[string][]byte{"assets/theme.css": []byte(canonicalCSS)},
			where: "assets/theme.css:3",
			found: true,
		},
		{
			name:  "layer statement",
			id:    "css.at.layer",
			files: map[string][]byte{"assets/theme.css": []byte(canonicalCSS)},
			where: "assets/theme.css:2",
			found: true,
		},
		{
			name:  "font-face",
			id:    "css.at.font-face",
			files: map[string][]byte{"assets/theme.css": []byte(canonicalCSS)},
			where: "assets/theme.css:1",
			found: true,
		},
		{
			name:  "keyframes",
			id:    "css.at.keyframes",
			files: map[string][]byte{"assets/theme.css": []byte(canonicalCSS)},
			where: "assets/theme.css:4",
			found: true,
		},
		{
			name:  "layer block",
			id:    "css.at.layer",
			files: map[string][]byte{"assets/theme.css": []byte("@media print{\n@layer x{@scope (umono-root){a{color:red}}}}\n")},
			where: "assets/theme.css:2",
			found: true,
		},
		{
			name:  "no media",
			id:    "css.at.media",
			files: map[string][]byte{"assets/theme.css": []byte("@scope (umono-root){a{color:red}}\n")},
			where: "",
			found: false,
		},
		{
			name:  "no css",
			id:    "css.at.media",
			files: map[string][]byte{"built.toml": []byte("x")},
			where: "",
			found: false,
		},
		{
			name:  "unparsable",
			id:    "css.at.media",
			files: map[string][]byte{"assets/theme.css": []byte("@media print{")},
			where: "",
			found: false,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := feature(t, c.id)
			where, found := f.Detect(Package{Files: c.files})
			if where != c.where || found != c.found {
				t.Errorf("Detect(%s) = (%q, %v), want (%q, %v)", c.id, where, found, c.where, c.found)
			}
		})
	}
}

func TestDetectFeatureFloorUnreadableCSS(t *testing.T) {
	_, _, err := DetectFeatureFloor(Package{Files: map[string][]byte{"assets/theme.css": []byte("a{")}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "assets/theme.css") {
		t.Errorf("error %q does not contain %q", err.Error(), "assets/theme.css")
	}
}
