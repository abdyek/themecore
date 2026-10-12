package themecore

import (
	"fmt"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// valueFunctions is the value function dictionary of v0.9 (themecore, Karar
// 18; comment: closed, a function added later is a Feature row
// css.function.<name>).
var valueFunctions = map[string]struct{}{
	"calc":                      {},
	"min":                       {},
	"max":                       {},
	"clamp":                     {},
	"round":                     {},
	"mod":                       {},
	"rem":                       {},
	"abs":                       {},
	"sign":                      {},
	"sin":                       {},
	"cos":                       {},
	"tan":                       {},
	"asin":                      {},
	"acos":                      {},
	"atan":                      {},
	"atan2":                     {},
	"pow":                       {},
	"sqrt":                      {},
	"hypot":                     {},
	"log":                       {},
	"exp":                       {},
	"var":                       {},
	"counter":                   {},
	"counters":                  {},
	"attr":                      {},
	"rgb":                       {},
	"rgba":                      {},
	"hsl":                       {},
	"hsla":                      {},
	"hwb":                       {},
	"lab":                       {},
	"lch":                       {},
	"oklab":                     {},
	"oklch":                     {},
	"color":                     {},
	"color-mix":                 {},
	"light-dark":                {},
	"url":                       {},
	"src":                       {},
	"image-set":                 {},
	"linear-gradient":           {},
	"radial-gradient":           {},
	"conic-gradient":            {},
	"repeating-linear-gradient": {},
	"repeating-radial-gradient": {},
	"repeating-conic-gradient":  {},
	"translate":                 {},
	"translatex":                {},
	"translatey":                {},
	"translatez":                {},
	"translate3d":               {},
	"scale":                     {},
	"scalex":                    {},
	"scaley":                    {},
	"scalez":                    {},
	"scale3d":                   {},
	"rotate":                    {},
	"rotatex":                   {},
	"rotatey":                   {},
	"rotatez":                   {},
	"rotate3d":                  {},
	"skew":                      {},
	"skewx":                     {},
	"skewy":                     {},
	"matrix":                    {},
	"matrix3d":                  {},
	"perspective":               {},
	"blur":                      {},
	"brightness":                {},
	"contrast":                  {},
	"drop-shadow":               {},
	"grayscale":                 {},
	"hue-rotate":                {},
	"invert":                    {},
	"opacity":                   {},
	"saturate":                  {},
	"sepia":                     {},
	"inset":                     {},
	"circle":                    {},
	"ellipse":                   {},
	"polygon":                   {},
	"path":                      {},
	"cubic-bezier":              {},
	"steps":                     {},
	"linear":                    {},
	"repeat":                    {},
	"minmax":                    {},
	"fit-content":               {},
}

// fontFaceSrcFunctions is the set of functions allowed only in the src
// descriptor of @font-face. They are not allowed in any other declaration
// value.
var fontFaceSrcFunctions = map[string]struct{}{
	"local":  {},
	"format": {},
	"tech":   {},
}

// deniedProperties is the set of property names that are never allowed in
// any declaration list.
var deniedProperties = map[string]struct{}{
	"behavior":     {},
	"-moz-binding": {},
}

// declContext marks the kind of declaration list a declaration belongs to.
// The context drives a few value rules, in particular the font-face src
// check and the font-family reserved-name check.
type declContext int

const (
	// declStyle is the declaration list of a style rule.
	declStyle declContext = iota
	// declFontFace is the declaration list of an @font-face block.
	declFontFace
	// declKeyframe is the declaration list of a @keyframes rule entry.
	declKeyframe
)

// reservedName reports whether s uses a reserved umono- or compono- prefix.
// The check is ASCII case-insensitive and matches both the non-custom
// prefix ("umono-", "compono-") and the custom-property prefix
// ("--umono-", "--compono-").
func reservedName(s string) bool {
	l := asciiLower(s)
	return strings.HasPrefix(l, "umono-") ||
		strings.HasPrefix(l, "compono-") ||
		strings.HasPrefix(l, "--umono-") ||
		strings.HasPrefix(l, "--compono-")
}

// stringContent returns the unescaped content of a CSS string token without
// the surrounding quotes. The token is expected to be a css.StringToken;
// any leading and trailing quote of the same kind are stripped before the
// escapes are resolved.
func stringContent(t cssToken) string {
	data := t.data
	if len(data) >= 2 && (data[0] == '"' || data[0] == '\'') && data[len(data)-1] == data[0] {
		data = data[1 : len(data)-1]
	}
	return cssUnescape(data)
}

// reservedError returns the canonical error for a reserved umono- or
// compono- prefix, anchored at line.
func reservedError(line int) error {
	return fmt.Errorf("theme.css:%d: the umono- and compono- prefixes are reserved", line)
}

// checkDecl checks the value of d against the declaration rules. ctx is the
// kind of declaration list d belongs to. The first failing rule produces
// the returned error; on success checkDecl returns nil.
//
// The rules are, in order:
//  1. d.name is in deniedProperties → "property <name> is not allowed".
//  2. d.name is a custom property (starts with "--") and uses a reserved
//     umono- or compono- prefix → "the umono- and compono- prefixes are
//     reserved".
//  3. Each value token: a function token must be in valueFunctions (or in
//     fontFaceSrcFunctions, and only when ctx is declFontFace and d.name is
//     "src"); a name of "attr" requires its first non-whitespace argument
//     to be an ident whose asciiLower(cssUnescape) is not "id"; an ident
//     token must not use a reserved umono- or compono- prefix.
//  4. When ctx is declFontFace and d.name is "font-family", every string
//     token must not use a reserved umono- or compono- prefix.
func checkDecl(d cssDecl, ctx declContext) error {
	if _, denied := deniedProperties[d.name]; denied {
		return fmt.Errorf("theme.css:%d: property %s is not allowed", d.line, d.name)
	}
	if strings.HasPrefix(d.name, "--") && reservedName(d.name) {
		return reservedError(d.line)
	}
	for i, t := range d.value {
		switch t.tt {
		case css.FunctionToken:
			data := t.data
			if len(data) > 0 && data[len(data)-1] == '(' {
				data = data[:len(data)-1]
			}
			name := asciiLower(cssUnescape(data))
			if _, isSrcFn := fontFaceSrcFunctions[name]; isSrcFn {
				if ctx != declFontFace || d.name != "src" {
					return fmt.Errorf("theme.css:%d: function %s() is only allowed in the src descriptor of @font-face", t.line, name)
				}
			} else if _, ok := valueFunctions[name]; !ok {
				return fmt.Errorf("theme.css:%d: function %s() is not allowed", t.line, name)
			}
			if name == "attr" {
				j := i + 1
				for j < len(d.value) && d.value[j].tt == css.WhitespaceToken {
					j++
				}
				if j >= len(d.value) || d.value[j].tt != css.IdentToken {
					return fmt.Errorf("theme.css:%d: attr() must name an attribute other than id", t.line)
				}
				if asciiLower(cssUnescape(d.value[j].data)) == "id" {
					return fmt.Errorf("theme.css:%d: attr() must name an attribute other than id", t.line)
				}
			}
		case css.IdentToken:
			if reservedName(cssUnescape(t.data)) {
				return reservedError(t.line)
			}
		}
	}
	if ctx == declFontFace && d.name == "font-family" {
		for _, t := range d.value {
			if t.tt == css.StringToken {
				if reservedName(stringContent(t)) {
					return reservedError(t.line)
				}
			}
		}
	}
	return nil
}

// checkKeyframesName checks the name of @keyframes: one ident or string,
// without a reserved prefix.
func checkKeyframesName(prelude []cssToken, line int) error {
	toks := trimSpace(prelude)
	if len(toks) != 1 {
		return fmt.Errorf("theme.css:%d: invalid @keyframes name", line)
	}
	t := toks[0]
	var name string
	switch t.tt {
	case css.IdentToken:
		name = cssUnescape(t.data)
	case css.StringToken:
		name = stringContent(t)
	default:
		return fmt.Errorf("theme.css:%d: invalid @keyframes name", line)
	}
	if reservedName(name) {
		return reservedError(line)
	}
	return nil
}

// checkLayerNames checks the names of an @layer prelude: a comma-separated
// list of names (one name at most for a block), each an ident optionally
// followed by "." and an ident, repeated, without whitespace. The first
// segment must not have a reserved prefix.
func checkLayerNames(prelude []cssToken, block bool, line int) error {
	if len(trimSpace(prelude)) == 0 {
		return nil
	}
	items := splitTopComma(prelude)
	if block && len(items) > 1 {
		return fmt.Errorf("theme.css:%d: invalid @layer name", line)
	}
	for _, raw := range items {
		toks := trimSpace(raw)
		if len(toks) == 0 || toks[0].tt != css.IdentToken {
			return fmt.Errorf("theme.css:%d: invalid @layer name", line)
		}
		if reservedName(cssUnescape(toks[0].data)) {
			return reservedError(line)
		}
		i := 1
		for i < len(toks) {
			if toks[i].tt != css.DelimToken || len(toks[i].data) != 1 || toks[i].data[0] != '.' {
				return fmt.Errorf("theme.css:%d: invalid @layer name", line)
			}
			if i+1 >= len(toks) || toks[i+1].tt != css.IdentToken {
				return fmt.Errorf("theme.css:%d: invalid @layer name", line)
			}
			i += 2
		}
	}
	return nil
}
