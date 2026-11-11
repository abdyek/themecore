package themecore

import (
	"fmt"
	"strings"
	"testing"
)

// valid author stylesheets; every one must pass transform, and every transform
// output must pass verify, themecore Karar 11
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

func TestTransformOutputs(t *testing.T) {
	for i, css := range validStylesheets {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out, err := transform([]byte(css))
			if err != nil {
				t.Fatalf("transform error: %v", err)
			}
			if err := verify(out); err != nil {
				t.Fatalf("verify error on transformed output: %v", err)
			}
			_, err = transform(out)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", "@scope is not allowed")
			}
			if !strings.Contains(err.Error(), "@scope is not allowed") {
				t.Fatalf("expected error containing %q, got %v", "@scope is not allowed", err)
			}
		})
	}
}
