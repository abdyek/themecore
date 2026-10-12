package themecore

import (
	"strings"
	"testing"

	"github.com/tdewolff/parse/v2/css"
)

func dumpText(tokens []cssToken) string {
	var sb strings.Builder
	for _, t := range tokens {
		if t.tt == css.WhitespaceToken {
			sb.WriteByte(' ')
		} else {
			sb.Write(t.data)
		}
	}
	return sb.String()
}

func dumpCSS(rules []cssRule) string {
	var sb strings.Builder
	dumpRules(&sb, rules, 0)
	return sb.String()
}

func dumpRules(sb *strings.Builder, rules []cssRule, indent int) {
	prefix := strings.Repeat(" ", indent)
	for _, r := range rules {
		if r.at == "" {
			sb.WriteString(prefix)
			sb.WriteString("rule ")
			sb.WriteString(dumpText(r.prelude))
			sb.WriteByte('\n')
			dumpDecls(sb, r.decls, indent+2)
			continue
		}
		sb.WriteString(prefix)
		sb.WriteByte('@')
		sb.WriteString(r.at)
		if len(r.prelude) > 0 {
			sb.WriteByte(' ')
			sb.WriteString(dumpText(r.prelude))
		}
		if r.block {
			sb.WriteByte('\n')
			dumpRules(sb, r.rules, indent+2)
			dumpDecls(sb, r.decls, indent+2)
		} else {
			sb.WriteString(";\n")
		}
	}
}

func dumpDecls(sb *strings.Builder, decls []cssDecl, indent int) {
	prefix := strings.Repeat(" ", indent)
	for _, d := range decls {
		sb.WriteString(prefix)
		sb.WriteString("decl ")
		sb.WriteString(d.name)
		sb.WriteString(": ")
		sb.WriteString(dumpText(d.value))
		if d.important {
			sb.WriteString(" !important")
		}
		sb.WriteByte('\n')
	}
}

func TestParseCSS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "style rule",
			input: `h1 { color: red; margin: 0 auto }`,
			want:  "rule h1\n  decl color: red\n  decl margin: 0 auto\n",
		},
		{
			name:  "important",
			input: `a{color:red!important}`,
			want:  "rule a\n  decl color: red !important\n",
		},
		{
			name:  "important with spaces",
			input: `a{color: red ! IMPORTANT ;}`,
			want:  "rule a\n  decl color: red !important\n",
		},
		{
			name:  "custom property",
			input: `a{--Main-Color: #fff}`,
			want:  "rule a\n  decl --Main-Color: #fff\n",
		},
		{
			name:  "property case",
			input: `a{COLOR:red}`,
			want:  "rule a\n  decl color: red\n",
		},
		{
			name:  "empty declarations",
			input: `a{;;color:red;;}`,
			want:  "rule a\n  decl color: red\n",
		},
		{
			name:  "media",
			input: `@media (min-width: 48rem) { :scope > nav { display: flex } }`,
			want:  "@media (min-width: 48rem)\n  rule :scope > nav\n    decl display: flex\n",
		},
		{
			name:  "media without block",
			input: `@media print;`,
			want:  "@media print;\n",
		},
		{
			name:  "layer statement",
			input: `@layer base, components;`,
			want:  "@layer base, components;\n",
		},
		{
			name:  "layer block",
			input: `@layer x { .card { padding: 1rem } }`,
			want:  "@layer x\n  rule .card\n    decl padding: 1rem\n",
		},
		{
			name:  "font-face",
			input: `@font-face { font-family: "acme"; src: url(fonts/a.woff2) format("woff2") }`,
			want:  "@font-face\n  decl font-family: \"acme\"\n  decl src: url(fonts/a.woff2) format(\"woff2\")\n",
		},
		{
			name:  "keyframes",
			input: `@keyframes spin { from { opacity: 0 } 50% { opacity: .5 } }`,
			want:  "@keyframes spin\n  rule from\n    decl opacity: 0\n  rule 50%\n    decl opacity: .5\n",
		},
		{
			name:  "scope",
			input: `@scope (umono-root) { h1 { color: red } }`,
			want:  "@scope (umono-root)\n  rule h1\n    decl color: red\n",
		},
		{
			name:  "supports nested media",
			input: `@supports (display: grid) { @media print { a { color: red } } }`,
			want:  "@supports (display: grid)\n  @media print\n    rule a\n      decl color: red\n",
		},
		{
			name:  "escaped at-rule name",
			input: `@\6d edia print { a { color: red } }`,
			want:  "@media print\n  rule a\n    decl color: red\n",
		},
		{
			name:  "upper case at-rule name",
			input: `@MEDIA print { a { color: red } }`,
			want:  "@media print\n  rule a\n    decl color: red\n",
		},
		{
			name:  "comments",
			input: `/* c */ a /* x */ b { /* y */ color: /* z */ red }`,
			want:  "rule a b\n  decl color: red\n",
		},
		{
			name:  "whitespace runs",
			input: "a \n\t b{color:red}",
			want:  "rule a b\n  decl color: red\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := parseCSS([]byte(tt.input))
			if err != nil {
				t.Fatalf("parseCSS(%q) error: %v", tt.input, err)
			}
			got := dumpCSS(rules)
			if got != tt.want {
				t.Errorf("dumpCSS = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCSSUnescape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", `abc`, "abc"},
		{"hex name", `\6d edia`, "media"},
		{"hex start", `\69mport`, "import"},
		{"escaped colon", `\:root`, ":root"},
		{"hex middle", `u\72l`, "url"},
		{"full hex", `\000075mono`, "umono"},
		{"trailing backslash", "a\\", "a\uFFFD"},
		{"null byte", `\0`, "\uFFFD"},
		{"out of range", `\110000`, "\uFFFD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cssUnescape([]byte(tt.input))
			if got != tt.want {
				t.Errorf("cssUnescape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCSSRejects(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "unclosed block",
			input: []byte(`a { color: red`),
			want:  "theme.css:1: unclosed block",
		},
		{
			name:  "unclosed at-rule block",
			input: []byte(`@media print { a { color: red }`),
			want:  "theme.css:1: unclosed block",
		},
		{
			name:  "stray close brace",
			input: []byte(`a { color: red } }`),
			want:  "theme.css:1: unexpected }",
		},
		{
			name:  "error on line 3",
			input: []byte("a { color: red }\n\nb { color: red } }"),
			want:  "theme.css:3: unexpected }",
		},
		{
			name:  "bad string",
			input: []byte("a { content: \"x\n\" }"),
			want:  "bad string",
		},
		{
			name:  "bad url",
			input: []byte(`a { background: url(a b.png) }`),
			want:  "bad url",
		},
		{
			name:  "cdo",
			input: []byte(`<!-- a { color: red } -->`),
			want:  "<!-- and --> are not allowed",
		},
		{
			name:  "cdc in value",
			input: []byte(`a { color: red --> }`),
			want:  "<!-- and --> are not allowed",
		},
		{
			name:  "no colon",
			input: []byte(`a { color red }`),
			want:  "expected : after property name",
		},
		{
			name:  "not a property",
			input: []byte(`a { 1px: red }`),
			want:  "expected a property name",
		},
		{
			name:  "empty value",
			input: []byte(`a { color: ; }`),
			want:  "empty value",
		},
		{
			name:  "important only",
			input: []byte(`a { color: !important }`),
			want:  "empty value",
		},
		{
			name:  "block in value",
			input: []byte(`a { --x: { b: c } }`),
			want:  "a block is not allowed here",
		},
		{
			name:  "nested rule",
			input: []byte(`a { b { color: red } }`),
			want:  "a block is not allowed here",
		},
		{
			name:  "empty selector",
			input: []byte(`{ color: red }`),
			want:  "empty prelude",
		},
		{
			name:  "unbalanced parenthesis",
			input: []byte(`a:is(b { color: red }`),
			want:  "unbalanced",
		},
		{
			name:  "unbalanced bracket in value",
			input: []byte(`a { grid-template-areas: [x }`),
			want:  "unbalanced",
		},
		{
			name:  "rule without block",
			input: []byte(`a;`),
			want:  "expected {",
		},
		{
			name:  "eof after selector",
			input: []byte(`a`),
			want:  "expected {",
		},
		{
			name:  "unknown at-rule",
			input: []byte(`@import "a.css";`),
			want:  "@import is not allowed",
		},
		{
			name:  "escaped unknown at-rule",
			input: []byte(`@\69mport "a.css";`),
			want:  "@import is not allowed",
		},
		{
			name:  "charset",
			input: []byte(`@charset "UTF-8"; a { color: red }`),
			want:  "@charset is not allowed",
		},
		{
			name:  "font-face with rules",
			input: []byte(`@font-face { a { color: red } }`),
			want:  "a block is not allowed here",
		},
		{
			name:  "comment merges idents",
			input: []byte(`umono/**/-root { color: red }`),
			want:  "comment separates tokens",
		},
		{
			name:  "comment merges in value",
			input: []byte(`a { font-family: umono/**/-sans }`),
			want:  "comment separates tokens",
		},
		{
			name:  "invalid utf-8",
			input: append(append([]byte(`a { content: "`), 0xff), []byte(`x }`)...),
			want:  "not valid UTF-8",
		},
		{
			name:  "nul byte",
			input: []byte("a { color: red\x00 }"),
			want:  "NUL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCSS(tt.input)
			if err == nil {
				t.Fatalf("parseCSS(%q) error = nil, want containing %q", tt.input, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("parseCSS(%q) error = %q, want containing %q", tt.input, err.Error(), tt.want)
			}
		})
	}
}
