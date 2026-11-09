package themecore

import (
	"errors"
	"fmt"
)

// transform turns the theme author's assets/theme.css into the canonical CSS
// that goes into the package (themecore, CSS Scoping). It drops a leading
// byte order mark and a leading @charset "UTF-8"; statement, parses the CSS
// strictly, checks the author grammar and serializes the canonical grammar,
// in which every style rule is inside an @scope (umono-root) block. The
// output always passes verify; if it does not, transform returns an error
// instead of the output, because that is a bug in themecore (Karar 11).
func transform(css []byte) (out []byte, err error) {
	b := css
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}
	if len(b) >= len(`@charset "UTF-8";`) && string(b[:len(`@charset "UTF-8";`)]) == `@charset "UTF-8";` {
		b = b[len(`@charset "UTF-8";`):]
	}
	rules, err := parseCSS(b)
	if err != nil {
		return nil, err
	}
	if err := checkGrammar(rules, false); err != nil {
		return nil, err
	}
	out = serializeCSS(rules)
	if err := verify(out); err != nil {
		return nil, fmt.Errorf("themecore: the canonical CSS does not pass verify, this is a bug: %w", err)
	}
	return out, nil
}

// verify checks that css, the assets/theme.css of a package, is in the
// canonical grammar (themecore, Karar 9). It is structural: it does not
// compare bytes with transform's output, so the serialization format is not
// part of the Theme API (Karar 10).
func verify(css []byte) error {
	if len(css) >= 3 && css[0] == 0xEF && css[1] == 0xBB && css[2] == 0xBF {
		return errors.New("theme.css: a byte order mark is not allowed")
	}
	if cssHasComment(css) {
		return errors.New("theme.css: comments are not allowed")
	}
	rules, err := parseCSS(css)
	if err != nil {
		return err
	}
	return checkGrammar(rules, true)
}
