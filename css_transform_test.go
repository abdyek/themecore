package themecore

import (
	"strings"
	"testing"
)

func TestTransformExample(t *testing.T) {
	in := `@charset "UTF-8";
@font-face { font-family: "acme-inter"; src: url(fonts/inter.woff2) format("woff2"); }
@layer base, components;

:scope.layout-post { max-width: 60rem; }
h1 { font-family: "acme-inter", sans-serif; }

@media (min-width: 48rem) {
  :scope > nav { display: flex; }
}
@layer components {
  .card { padding: 1rem; }
}
`
	want := `@font-face{font-family:"acme-inter";src:url(fonts/inter.woff2) format("woff2")}
@layer base,components;
@scope (umono-root){:scope.layout-post{max-width:60rem}h1{font-family:"acme-inter",sans-serif}}
@media (min-width:48rem){@scope (umono-root){:scope>nav{display:flex}}}
@layer components{@scope (umono-root){.card{padding:1rem}}}
`
	out, err := Transform([]byte(in))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("mismatch:\n got: %q\nwant: %q", string(out), want)
	}
}

func TestTransform(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "split by font-face",
			in:   `a{color:red} @font-face{font-family:x} b{color:blue}`,
			want: "@scope (umono-root){a{color:red}}\n@font-face{font-family:x}\n@scope (umono-root){b{color:blue}}\n",
		},
		{
			name: "nested blocks",
			in:   `@media print { @supports (display: grid) { a { color: red } } b { color: blue } }`,
			want: "@media print{@supports (display:grid){@scope (umono-root){a{color:red}}}@scope (umono-root){b{color:blue}}}\n",
		},
		{
			name: "bom and charset",
			in:   "\xEF\xBB\xBF" + `@charset "UTF-8";a{color:red}`,
			want: "@scope (umono-root){a{color:red}}\n",
		},
		{
			name: "comments",
			in:   `/* x */ a /* y */ { color: /* z */ red }`,
			want: "@scope (umono-root){a{color:red}}\n",
		},
		{
			name: "important",
			in:   `a { color: red ! important }`,
			want: "@scope (umono-root){a{color:red!important}}\n",
		},
		{
			name: "descendant and child",
			in:   `a  b > c , d  e { margin: 0  auto }`,
			want: "@scope (umono-root){a b>c,d e{margin:0 auto}}\n",
		},
		{
			name: "calc keeps spaces",
			in:   `a { width: calc( 100% - 2 * 1rem ) }`,
			want: "@scope (umono-root){a{width:calc(100% - 2 * 1rem)}}\n",
		},
		{
			name: "custom property keeps its value",
			in:   `a { --x: foo: bar , baz }`,
			want: "@scope (umono-root){a{--x:foo: bar , baz}}\n",
		},
		{
			name: "empty rule",
			in:   `a {}`,
			want: "@scope (umono-root){a{}}\n",
		},
		{
			name: "keyframes",
			in:   `@keyframes spin { from { opacity: 0 } 50%, to { opacity: 1 } }`,
			want: "@keyframes spin{from{opacity:0}50%,to{opacity:1}}\n",
		},
		{
			name: "anonymous layer",
			in:   `@layer { a { color: red } }`,
			want: "@layer{@scope (umono-root){a{color:red}}}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Transform([]byte(tc.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out) != tc.want {
				t.Errorf("mismatch:\n got: %q\nwant: %q", string(out), tc.want)
			}
		})
	}
}

func TestTransformRejectsGrammar(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "scope",
			in:   `@scope (umono-root) { a { color: red } }`,
			want: "theme.css:1: @scope is not allowed",
		},
		{
			name: "font-face in media",
			in:   `@media print { @font-face { font-family: x } }`,
			want: "@font-face is only allowed at the top level",
		},
		{
			name: "keyframes in layer",
			in:   `@layer x { @keyframes a { from { opacity: 0 } } }`,
			want: "@keyframes is only allowed at the top level",
		},
		{
			name: "layer statement in supports",
			in:   `@supports (display: grid) { @layer a; }`,
			want: "@layer statements are only allowed at the top level",
		},
		{
			name: "media without block",
			in:   `@media print;`,
			want: "@media must have a block",
		},
		{
			name: "supports without block",
			in:   `@supports (display: grid);`,
			want: "@supports must have a block",
		},
		{
			name: "empty layer statement",
			in:   `@layer;`,
			want: "@layer statement must name a layer",
		},
		{
			name: "font-face prelude",
			in:   `@font-face x { font-family: y }`,
			want: "@font-face takes no prelude",
		},
		{
			name: "font-face without block",
			in:   `@font-face;`,
			want: "@font-face must have a block",
		},
		{
			name: "keyframes without name",
			in:   `@keyframes { from { opacity: 0 } }`,
			want: "@keyframes must have a name",
		},
		{
			name: "keyframes without block",
			in:   `@keyframes a;`,
			want: "@keyframes must have a block",
		},
		{
			name: "keyframe selector",
			in:   `@keyframes a { b { opacity: 0 } }`,
			want: "keyframe selector must be from, to or a percentage",
		},
		{
			name: "important in keyframes",
			in:   `@keyframes a { from { opacity: 0 !important } }`,
			want: "!important is not allowed in @keyframes",
		},
		{
			name: "important in font-face",
			in:   `@font-face { font-family: x !important }`,
			want: "!important is not allowed in @font-face",
		},
		{
			name: "lower case charset",
			in:   `@charset "utf-8"; a { color: red }`,
			want: "@charset is not allowed",
		},
		{
			name: "charset later",
			in:   `a { color: red } @charset "UTF-8";`,
			want: "@charset is not allowed",
		},
		{
			name: "line number",
			in:   "a { color: red }\n\n@media print;",
			want: "theme.css:3: @media must have a block",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Transform([]byte(tc.in))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{
			name: "empty",
			in:   ``,
		},
		{
			name: "transform output",
			in:   `@scope (umono-root){a{color:red}}` + "\n" + `@media print{@scope (umono-root){b{color:blue}}}` + "\n",
		},
		{
			name: "other formatting",
			in:   "@scope (umono-root) {\n  a { color: red; }\n}\n@layer base, components;\n",
		},
		{
			name: "at-rules",
			in:   `@font-face{font-family:x}@keyframes a{from{opacity:0}}@layer a,b;@supports (display:grid){@scope (umono-root){a{display:grid}}}`,
		},
		{
			name: "two scope blocks",
			in:   `@scope (umono-root){a{color:red}}@scope (umono-root){b{color:blue}}`,
		},
		{
			name: "empty scope block",
			in:   `@scope (umono-root){}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Verify([]byte(tc.in)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestVerifyRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "top-level style rule",
			in:   `a{color:red}`,
			want: "theme.css:1: style rule outside @scope (umono-root)",
		},
		{
			name: "style rule in media",
			in:   `@media print{a{color:red}}`,
			want: "style rule outside @scope (umono-root)",
		},
		{
			name: "scope with limit",
			in:   `@scope (umono-root) to (p){a{color:red}}`,
			want: "@scope prelude must be (umono-root)",
		},
		{
			name: "scope of html",
			in:   `@scope (html){a{color:red}}`,
			want: "@scope prelude must be (umono-root)",
		},
		{
			name: "scope without prelude",
			in:   `@scope {a{color:red}}`,
			want: "@scope prelude must be (umono-root)",
		},
		{
			name: "escaped scope root",
			in:   `@scope (\75mono-root){a{color:red}}`,
			want: "@scope prelude must be (umono-root)",
		},
		{
			name: "at-rule in scope",
			in:   `@scope (umono-root){@media print{a{color:red}}}`,
			want: "@scope may contain only style rules",
		},
		{
			name: "scope without block",
			in:   `@scope (umono-root);`,
			want: "@scope must have a block",
		},
		{
			name: "comment",
			in:   `/* x */@scope (umono-root){a{color:red}}`,
			want: "theme.css: comments are not allowed",
		},
		{
			name: "bom",
			in:   "\xEF\xBB\xBF" + `@scope (umono-root){a{color:red}}`,
			want: "theme.css: a byte order mark is not allowed",
		},
		{
			name: "charset",
			in:   `@charset "UTF-8";@scope (umono-root){a{color:red}}`,
			want: "@charset is not allowed",
		},
		{
			name: "font-face in media",
			in:   `@media print{@font-face{font-family:x}}`,
			want: "@font-face is only allowed at the top level",
		},
		{
			name: "parse error",
			in:   `@scope (umono-root){a{color:red}`,
			want: "unclosed block",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify([]byte(tc.in))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
