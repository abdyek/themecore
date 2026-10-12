package themecore

import (
	"fmt"
	"unicode"

	"github.com/tdewolff/parse/v2/css"
)

// textProperties is the set of declaration names that may add text to the
// accessibility tree or a generated-content box: "content", "quotes",
// "list-style-type", "list-style".
var textProperties = map[string]struct{}{
	"content":         {},
	"quotes":          {},
	"list-style-type": {},
	"list-style":      {},
}

// imageFunctions is the set of CSS function names that render an image:
// "url", "src", "image-set", "linear-gradient", "radial-gradient",
// "conic-gradient", "repeating-linear-gradient", "repeating-radial-gradient",
// "repeating-conic-gradient".
var imageFunctions = map[string]struct{}{
	"url":                       {},
	"src":                       {},
	"image-set":                 {},
	"linear-gradient":           {},
	"radial-gradient":           {},
	"conic-gradient":            {},
	"repeating-linear-gradient": {},
	"repeating-radial-gradient": {},
	"repeating-conic-gradient":  {},
}

// checkGeneratedText reports whether d obeys the "no generated text" rule
// of themecore (Üretilen Metin Kuralı: theme CSS cannot generate words or
// add text to the accessibility tree). When d.name is not in textProperties,
// checkGeneratedText returns nil. The rules are, in order:
//
//  1. a function token named "var" → "var() is not allowed in <name>";
//  2. a string token whose innermost enclosing function is not "url",
//     "src" or "image-set" must not contain a rune for which
//     unicode.IsLetter is true → "strings in <name> must not contain
//     letters";
//  3. when d.name is "content", a string or an image at depth 0 needs an
//     alternative text after a "/" delimiter; the alternative text must
//     not itself contain a non-empty string.
//
// The walk keeps a stack of enclosing function names: a function token
// pushes asciiLower(cssUnescape(name without "(")), a "(" pushes "", and a
// ")" pops. All errors are anchored at d.line.
func checkGeneratedText(d cssDecl) error {
	if _, ok := textProperties[d.name]; !ok {
		return nil
	}
	var stack []string
	for _, t := range d.value {
		switch t.tt {
		case css.FunctionToken:
			data := t.data
			if len(data) > 0 && data[len(data)-1] == '(' {
				data = data[:len(data)-1]
			}
			name := asciiLower(cssUnescape(data))
			if name == "var" {
				return fmt.Errorf("theme.css:%d: var() is not allowed in %s", d.line, d.name)
			}
			stack = append(stack, name)
		case css.LeftParenthesisToken:
			stack = append(stack, "")
		case css.RightParenthesisToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case css.StringToken:
			innermost := ""
			if len(stack) > 0 {
				innermost = stack[len(stack)-1]
			}
			if innermost != "url" && innermost != "src" && innermost != "image-set" {
				for _, r := range stringContent(t) {
					if unicode.IsLetter(r) {
						return fmt.Errorf("theme.css:%d: strings in %s must not contain letters", d.line, d.name)
					}
				}
			}
		}
	}
	if d.name == "content" {
		main, alt, hasAlt := splitContentAlt(d.value)
		if needsAltText(main) {
			if !hasAlt || len(nonWhitespaceTokens(alt)) == 0 {
				return fmt.Errorf("theme.css:%d: content with a string or an image needs alternative text", d.line)
			}
		}
		if hasAlt && hasNonEmptyString(alt) {
			return fmt.Errorf("theme.css:%d: the alternative text of content must not contain a non-empty string", d.line)
		}
	}
	return nil
}

// splitContentAlt splits toks at the first "/" delim that is at depth 0
// (outside every function and parenthesis). It returns the slice before the
// delim, the slice after the delim, and whether a delim was found. The
// returned slices preserve the original depth relationships: a function
// token in main keeps its children in main.
func splitContentAlt(toks []cssToken) (main, alt []cssToken, found bool) {
	var stack []byte
	for i, t := range toks {
		switch t.tt {
		case css.FunctionToken, css.LeftParenthesisToken:
			stack = append(stack, ')')
		case css.RightParenthesisToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case css.DelimToken:
			if len(stack) == 0 && len(t.data) == 1 && t.data[0] == '/' {
				return toks[:i], toks[i+1:], true
			}
		}
	}
	return toks, nil, false
}

// needsAltText reports whether the "content" main part needs an alternative
// text. It does when the main part has, at depth 0, a non-empty string, a
// css.URLToken, or a function token whose name is in imageFunctions.
func needsAltText(toks []cssToken) bool {
	var stack []byte
	for _, t := range toks {
		switch t.tt {
		case css.FunctionToken:
			data := t.data
			if len(data) > 0 && data[len(data)-1] == '(' {
				data = data[:len(data)-1]
			}
			name := asciiLower(cssUnescape(data))
			if len(stack) == 0 {
				if _, ok := imageFunctions[name]; ok {
					return true
				}
			}
			stack = append(stack, ')')
		case css.LeftParenthesisToken:
			stack = append(stack, ')')
		case css.RightParenthesisToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case css.StringToken:
			if len(stack) == 0 && stringContent(t) != "" {
				return true
			}
		case css.URLToken:
			if len(stack) == 0 {
				return true
			}
		}
	}
	return false
}

// hasNonEmptyString reports whether toks contains, at any depth, a string
// token with a non-empty stringContent.
func hasNonEmptyString(toks []cssToken) bool {
	for _, t := range toks {
		if t.tt == css.StringToken && stringContent(t) != "" {
			return true
		}
	}
	return false
}

// nonWhitespaceTokens returns the tokens of toks that are not whitespace
// tokens, in their original order.
func nonWhitespaceTokens(toks []cssToken) []cssToken {
	out := make([]cssToken, 0, len(toks))
	for _, t := range toks {
		if t.tt != css.WhitespaceToken {
			out = append(out, t)
		}
	}
	return out
}
