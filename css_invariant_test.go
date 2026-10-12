package themecore

import (
	"bytes"
	"fmt"
	"testing"
)

// valid author stylesheets; every one must pass Transform, and every Transform
// output must pass Verify, themecore Karar 11
var validStylesheets = []string{
	`@media (min-width: 48rem) {
  @supports selector(::details-content) {
    nav > details > summary { display: none; }
    nav > details::details-content { content-visibility: visible; }
  }
}`,
	`@charset "UTF-8";
@font-face { font-family: "acme-inter"; src: url(fonts/inter.woff2) format("woff2"); }
@layer base, components;
:scope.layout-post { max-width: 60rem; }
h1 { font-family: "acme-inter", sans-serif; }
@media (min-width: 48rem) { :scope > nav { display: flex; } }
@layer components { .card { padding: 1rem; } }`,
	`:scope { display: grid; grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr)); } :scope > header, :scope > footer { grid-column: 1 / -1; }`,
	`@keyframes fade { from { opacity: 0 } to { opacity: 1 } } .card:hover { animation: fade .2s ease-in; }`,
	`@media print { @layer print { a::after { content: "" / ""; } } } @supports (display: grid) and (not (display: inline-grid)) { :scope:has(> aside) main { margin: 0; } }`,
	`li:nth-child(2n + 1 of .item):not(:last-child) { --gap: 1rem; margin-block-end: var(--gap); } p:lang(tr, "en"):dir(ltr)::first-line { font-weight: 600; }`,
	`@media (prefers-color-scheme: dark) and (min-resolution: 2dppx), print { :scope { color-scheme: dark; } }`,
}

func TestTransformOutputsVerify(t *testing.T) {
	for i, css := range validStylesheets {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out, err := Transform([]byte(css))
			if err != nil {
				t.Fatalf("Transform error: %v", err)
			}
			if err := Verify(out); err != nil {
				t.Fatalf("Verify error on transformed output: %v", err)
			}
		})
	}
}

func TestDetailsRecipe(t *testing.T) {
	src := validStylesheets[0]
	out, err := Transform([]byte(src))
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}
	want := "@media (min-width:48rem){@supports selector(::details-content){@scope (umono-root){nav>details>summary{display:none}nav>details::details-content{content-visibility:visible}}}}\n"
	if !bytes.Equal(out, []byte(want)) {
		t.Fatalf("unexpected output:\nwant: %q\ngot:  %q", want, out)
	}
}

func TestTransformIdempotentShape(t *testing.T) {
	for i, css := range validStylesheets {
		first, err := Transform([]byte(css))
		if err != nil {
			t.Fatalf("[%d] first Transform error: %v", i, err)
		}
		_, err2 := Transform(first)
		if bytes.Contains(first, []byte("@scope")) {
			if err2 == nil {
				t.Fatalf("[%d] expected error containing %q, got nil", i, "@scope is not allowed")
			}
			if !bytes.Contains([]byte(err2.Error()), []byte("@scope is not allowed")) {
				t.Fatalf("[%d] expected error containing %q, got %v", i, "@scope is not allowed", err2)
			}
			continue
		}
		if err2 != nil {
			t.Fatalf("[%d] expected success on second pass, got %v", i, err2)
		}
	}
}
