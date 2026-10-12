package themecore

import (
	"reflect"
	"testing"
)

const lintAttrs = `schema = "https://umono.dev/schemas/theme-attributes/v1"

[address."post"]
class = "layout-post"

[address."home"]
class = "layout-home shared"

[address."post IMAGE"]
class = "shared cover"`

const lintCSS = `@scope (umono-root){.layout-post h1{color:red}:scope.layout-post h2{color:red}.cover{color:red}.shared{color:red}:is(.layout-home) p{color:red}}
@media print{@scope (umono-root){article .layout-home,p{color:red}}}
`

func TestLintRootClass(t *testing.T) {
	p := Package{Files: map[string][]byte{
		"attributes.toml":  []byte(lintAttrs),
		"assets/theme.css": []byte(lintCSS),
	}}
	got := Lint(p)
	want := []Warning{
		{
			Message: ".layout-post h1 never matches: .layout-post is a root class, on <umono-root> only; use :scope.layout-post",
			Where:   ".layout-post h1",
		},
		{
			Message: "article .layout-home never matches: .layout-home is a root class, on <umono-root> only; use :scope.layout-home",
			Where:   "article .layout-home",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lint() = %v, want %v", got, want)
	}
}

func TestLintNothing(t *testing.T) {
	tests := []struct {
		name string
		p    Package
	}{
		{
			name: "only css",
			p:    Package{Files: map[string][]byte{"assets/theme.css": []byte(lintCSS)}},
		},
		{
			name: "only toml",
			p:    Package{Files: map[string][]byte{"attributes.toml": []byte(lintAttrs)}},
		},
		{
			name: "both ok",
			p: Package{Files: map[string][]byte{
				"attributes.toml":  []byte(lintAttrs),
				"assets/theme.css": []byte("@scope (umono-root){:scope.layout-post h1{color:red}}"),
			}},
		},
		{
			name: "bad toml",
			p: Package{Files: map[string][]byte{
				"attributes.toml":  []byte("x = "),
				"assets/theme.css": []byte(lintCSS),
			}},
		},
		{
			name: "bad css",
			p: Package{Files: map[string][]byte{
				"attributes.toml":  []byte(lintAttrs),
				"assets/theme.css": []byte("a{"),
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lint(tt.p)
			if len(got) != 0 {
				t.Errorf("Lint() = %v, want nil", got)
			}
		})
	}
}
