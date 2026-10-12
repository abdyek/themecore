package themecore

import _ "embed"

//go:embed fallback/theme.css
var fallbackSource []byte

var fallbackCSS = mustTransform(fallbackSource)

// mustTransform runs Transform on themecore's own fallback source. A
// failure is a bug in themecore and is caught by the tests.
func mustTransform(b []byte) []byte {
	out, err := Transform(b)
	if err != nil {
		panic(err)
	}
	return out
}

// FallbackSource returns the source of the fallback theme, the default
// look of pages without an active theme, in the author grammar. It is
// also the starting assets/theme.css of umono theme create.
func FallbackSource() []byte {
	out := make([]byte, len(fallbackSource))
	copy(out, fallbackSource)
	return out
}

// FallbackCSS returns the canonical CSS of the fallback theme: the
// Transform output of FallbackSource. Umono serves it at
// /_theme/fallback-{release}/theme.css.
func FallbackCSS() []byte {
	out := make([]byte, len(fallbackCSS))
	copy(out, fallbackCSS)
	return out
}
