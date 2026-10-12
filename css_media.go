// Package themecore implements the Umono theme engine. This file defines the
// v0.9 media-feature dictionary, the media types, accepted length and
// resolution units, and the helpers that validate a media feature's parenthesised
// content.
package themecore

import (
	"fmt"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// mfType categorises a media feature by the kind of value it accepts.
type mfType int

const (
	// mfLength is a media feature whose value is a length or the number zero.
	mfLength mfType = iota
	// mfRatio is a media feature whose value is a ratio of two non-negative numbers.
	mfRatio
	// mfResolution is a media feature whose value is a resolution.
	mfResolution
	// mfKeyword is a media feature whose value is one of a fixed set of idents.
	mfKeyword
)

// mediaFeature is the metadata for a single media feature: its value kind, the
// fixed keywords it accepts when its kind is mfKeyword, and whether the
// min-/max- prefixed forms and range syntax are allowed.
type mediaFeature struct {
	typ      mfType
	keywords []string
	ranged   bool
}

// mediaFeatures is the closed list of v0.9 media features; a name or unit added
// later is a Feature row css.media.<name>.
var mediaFeatures = map[string]mediaFeature{
	"width":                  {typ: mfLength, ranged: true},
	"height":                 {typ: mfLength, ranged: true},
	"aspect-ratio":           {typ: mfRatio, ranged: true},
	"resolution":             {typ: mfResolution, ranged: true},
	"orientation":            {typ: mfKeyword, keywords: []string{"portrait", "landscape"}},
	"hover":                  {typ: mfKeyword, keywords: []string{"none", "hover"}},
	"any-hover":              {typ: mfKeyword, keywords: []string{"none", "hover"}},
	"pointer":                {typ: mfKeyword, keywords: []string{"none", "coarse", "fine"}},
	"any-pointer":            {typ: mfKeyword, keywords: []string{"none", "coarse", "fine"}},
	"prefers-color-scheme":   {typ: mfKeyword, keywords: []string{"light", "dark"}},
	"prefers-reduced-motion": {typ: mfKeyword, keywords: []string{"no-preference", "reduce"}},
	"prefers-contrast":       {typ: mfKeyword, keywords: []string{"no-preference", "more", "less", "custom"}},
	"forced-colors":          {typ: mfKeyword, keywords: []string{"none", "active"}},
}

// mediaTypes is the set of supported media types in v0.9.
var mediaTypes = map[string]bool{
	"all":    true,
	"screen": true,
	"print":  true,
}

// lengthUnits is the set of CSS length units accepted by v0.9 media queries,
// including the sv-, lv-, and dv- prefixed viewport-relative units.
var lengthUnits = map[string]bool{
	"px": true, "em": true, "rem": true, "ch": true, "ex": true,
	"vw": true, "vh": true, "vi": true, "vb": true, "vmin": true, "vmax": true,
	"svw": true, "svh": true, "svi": true, "svb": true, "svmin": true, "svmax": true,
	"lvw": true, "lvh": true, "lvi": true, "lvb": true, "lvmin": true, "lvmax": true,
	"dvw": true, "dvh": true, "dvi": true, "dvb": true, "dvmin": true, "dvmax": true,
}

// resolutionUnits is the set of CSS resolution units accepted by v0.9 media
// queries.
var resolutionUnits = map[string]bool{
	"dppx": true,
	"dpi":  true,
	"x":    true,
}

// dimensionUnit returns the unit portion of the DimensionToken t, with the
// number prefix (optional sign, digits, optional fractional part, optional
// exponent) stripped. The token data is unescaped and ASCII-lowered before
// parsing, so the result is always in canonical lower-case form.
func dimensionUnit(t cssToken) string {
	s := asciiLower(cssUnescape(t.data))
	i := 0
	n := len(s)
	if i < n && (s[i] == '+' || s[i] == '-') {
		i++
	}
	if i >= n || !isCSSDecDigit(s[i]) {
		return s
	}
	for i < n && isCSSDecDigit(s[i]) {
		i++
	}
	if i < n && s[i] == '.' {
		if i+1 < n && isCSSDecDigit(s[i+1]) {
			i++
			for i < n && isCSSDecDigit(s[i]) {
				i++
			}
		}
	}
	if i < n && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < n && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j < n && isCSSDecDigit(s[j]) {
			for j < n && isCSSDecDigit(s[j]) {
				j++
			}
			i = j
		}
	}
	return s[i:]
}

// isCSSDecDigit reports whether b is an ASCII decimal digit.
func isCSSDecDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// checkMediaValue reports whether toks form a valid value for f. The toks slice
// has no whitespace tokens.
func checkMediaValue(f mediaFeature, toks []cssToken) bool {
	switch f.typ {
	case mfLength:
		if len(toks) != 1 {
			return false
		}
		switch toks[0].tt {
		case css.DimensionToken:
			return lengthUnits[dimensionUnit(toks[0])]
		case css.NumberToken:
			return string(toks[0].data) == "0"
		}
		return false
	case mfResolution:
		if len(toks) != 1 || toks[0].tt != css.DimensionToken {
			return false
		}
		return resolutionUnits[dimensionUnit(toks[0])]
	case mfRatio:
		if len(toks) == 1 {
			if toks[0].tt != css.NumberToken {
				return false
			}
			return len(toks[0].data) > 0 && toks[0].data[0] != '-'
		}
		if len(toks) == 3 {
			if toks[0].tt != css.NumberToken || toks[2].tt != css.NumberToken {
				return false
			}
			if toks[1].tt != css.DelimToken || string(toks[1].data) != "/" {
				return false
			}
			return len(toks[0].data) > 0 && toks[0].data[0] != '-' &&
				len(toks[2].data) > 0 && toks[2].data[0] != '-'
		}
		return false
	case mfKeyword:
		if len(toks) != 1 || toks[0].tt != css.IdentToken {
			return false
		}
		kw := asciiLower(cssUnescape(toks[0].data))
		for _, k := range f.keywords {
			if kw == k {
				return true
			}
		}
		return false
	}
	return false
}

// lookupMediaFeature looks up name (already in lower case) in mediaFeatures.
// If name carries a "min-" or "max-" prefix and the rest names a ranged
// feature, the returned feature is that feature and prefixed is true. The
// boolean form rejects prefixed lookups; the named-value form allows them.
func lookupMediaFeature(name string) (f mediaFeature, prefixed, ok bool) {
	if v, found := mediaFeatures[name]; found {
		return v, false, true
	}
	if strings.HasPrefix(name, "min-") || strings.HasPrefix(name, "max-") {
		rest := name[4:]
		if v, found := mediaFeatures[rest]; found && v.ranged {
			return v, true, true
		}
	}
	return mediaFeature{}, false, false
}

// checkMediaFeature validates the parenthesised content of a media feature,
// with whitespace tokens already removed, and returns an error if invalid.
// The boolean form, the named-value form, and the range form are tried in
// order; the first that matches determines the error returned.
func checkMediaFeature(toks []cssToken, line int) error {
	if len(toks) == 1 && toks[0].tt == css.IdentToken {
		name := asciiLower(cssUnescape(toks[0].data))
		_, prefixed, ok := lookupMediaFeature(name)
		if !ok {
			return fmt.Errorf("theme.css:%d: unknown media feature %s", line, name)
		}
		if prefixed {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		return nil
	}
	if len(toks) >= 2 && toks[0].tt == css.IdentToken && toks[1].tt == css.ColonToken {
		name := asciiLower(cssUnescape(toks[0].data))
		f, _, ok := lookupMediaFeature(name)
		if !ok {
			return fmt.Errorf("theme.css:%d: unknown media feature %s", line, name)
		}
		if !checkMediaValue(f, toks[2:]) {
			return fmt.Errorf("theme.css:%d: invalid value of %s", line, name)
		}
		return nil
	}
	operands, operators := splitRangeOperands(toks)
	if len(operands) < 2 || len(operands) > 3 || len(operators) != len(operands)-1 {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	for _, op := range operands {
		if len(op) == 0 {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
	}
	nameIdx := -1
	var name string
	var f mediaFeature
	for i, op := range operands {
		if len(op) == 1 && op[0].tt == css.IdentToken {
			n := asciiLower(cssUnescape(op[0].data))
			ff, prefixed, ok := lookupMediaFeature(n)
			if ok && !prefixed && ff.ranged {
				if nameIdx != -1 {
					return fmt.Errorf("theme.css:%d: invalid media query", line)
				}
				nameIdx = i
				name = n
				f = ff
			}
		}
	}
	if nameIdx == -1 {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	for i, op := range operands {
		if i == nameIdx {
			continue
		}
		if !checkMediaValue(f, op) {
			return fmt.Errorf("theme.css:%d: invalid value of %s", line, name)
		}
	}
	if len(operands) == 3 {
		if nameIdx != 1 {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		op1 := string(operators[0].data)
		op2 := string(operators[1].data)
		isLeft := func(s string) bool { return s == "<" || s == "<=" }
		isRight := func(s string) bool { return s == ">" || s == ">=" }
		if !(isLeft(op1) && isLeft(op2)) && !(isRight(op1) && isRight(op2)) {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
	}
	return nil
}

// splitRangeOperands splits toks into the alternating operands and range
// operators that separate them. A "<=" or ">=" operator is two DelimTokens; a
// "<", ">", or "=" operator is one. Non-operator tokens are accumulated into
// the current operand.
func splitRangeOperands(toks []cssToken) (operands [][]cssToken, operators []cssToken) {
	current := []cssToken{}
	i := 0
	for i < len(toks) {
		if toks[i].tt == css.DelimToken {
			d := string(toks[i].data)
			if d == "<" || d == ">" || d == "=" {
				opData := d
				j := i + 1
				if (d == "<" || d == ">") && j < len(toks) && toks[j].tt == css.DelimToken && string(toks[j].data) == "=" {
					opData += "="
					j++
				}
				operands = append(operands, current)
				current = []cssToken{}
				operators = append(operators, cssToken{tt: css.DelimToken, data: []byte(opData)})
				i = j
				continue
			}
		}
		current = append(current, toks[i])
		i++
	}
	operands = append(operands, current)
	return operands, operators
}

// extractGroup returns the inner tokens of the parenthesised group whose
// opening parenthesis is ts[i]. Depth counts "(" and function tokens
// against ")" tokens. On a missing closing ")" ok is false.
func extractGroup(ts []cssToken, i int) (inner []cssToken, next int, ok bool) {
	depth := 1
	j := i + 1
	for j < len(ts) {
		switch ts[j].tt {
		case css.LeftParenthesisToken, css.FunctionToken:
			depth++
		case css.RightParenthesisToken:
			depth--
			if depth == 0 {
				return ts[i+1 : j], j + 1, true
			}
		}
		j++
	}
	return nil, j, false
}

// checkMediaQueryList checks an @media prelude, or with allowTypes false the
// media condition of a sizes value, against the Media Queries Level 4
// grammar and the media feature dictionary (themecore, Karar 16).
func checkMediaQueryList(toks []cssToken, allowTypes bool, line int) error {
	for _, p := range splitTopComma(toks) {
		p = trimSpace(p)
		if len(p) == 0 {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		filtered := make([]cssToken, 0, len(p))
		for _, t := range p {
			if t.tt != css.WhitespaceToken {
				filtered = append(filtered, t)
			}
		}
		if err := checkMediaQuery(filtered, allowTypes, line); err != nil {
			return err
		}
	}
	return nil
}

// checkMediaQuery validates a single media query, with whitespace tokens
// already removed. The keywords not, only, and, or are idents compared
// with asciiLower(cssUnescape).
func checkMediaQuery(ts []cssToken, allowTypes bool, line int) error {
	if len(ts) == 0 {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if ts[0].tt == css.LeftParenthesisToken {
		return checkMediaCondition(ts, true, line)
	}
	if ts[0].tt == css.IdentToken && asciiLower(cssUnescape(ts[0].data)) == "not" {
		if len(ts) >= 2 && ts[1].tt == css.LeftParenthesisToken {
			return checkMediaCondition(ts, true, line)
		}
	}
	i := 0
	if ts[i].tt == css.IdentToken {
		kw := asciiLower(cssUnescape(ts[i].data))
		if kw == "not" || kw == "only" {
			i++
		}
	}
	if i >= len(ts) {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if !allowTypes {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if ts[i].tt != css.IdentToken {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	typ := asciiLower(cssUnescape(ts[i].data))
	if !mediaTypes[typ] {
		return fmt.Errorf("theme.css:%d: unknown media type %s", line, typ)
	}
	i++
	if i == len(ts) {
		return nil
	}
	if !(ts[i].tt == css.IdentToken && asciiLower(cssUnescape(ts[i].data)) == "and") {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	i++
	if i == len(ts) {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	return checkMediaCondition(ts[i:], false, line)
}

// checkMediaCondition validates a media condition, with whitespace tokens
// already removed. The tokens must be consumed completely.
func checkMediaCondition(ts []cssToken, allowOr bool, line int) error {
	if len(ts) == 0 {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if ts[0].tt == css.IdentToken && asciiLower(cssUnescape(ts[0].data)) == "not" {
		if len(ts) < 2 || ts[1].tt != css.LeftParenthesisToken {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		inner, next, ok := extractGroup(ts, 1)
		if !ok {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		if next != len(ts) {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		return checkMediaInParens(inner, line)
	}
	if ts[0].tt != css.LeftParenthesisToken {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	inner, next, ok := extractGroup(ts, 0)
	if !ok {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if err := checkMediaInParens(inner, line); err != nil {
		return err
	}
	connector := ""
	for next < len(ts) {
		if ts[next].tt != css.IdentToken {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		kw := asciiLower(cssUnescape(ts[next].data))
		switch kw {
		case "and":
			if connector == "" {
				connector = "and"
			} else if connector != "and" {
				return fmt.Errorf("theme.css:%d: invalid media query", line)
			}
		case "or":
			if !allowOr {
				return fmt.Errorf("theme.css:%d: invalid media query", line)
			}
			if connector == "" {
				connector = "or"
			} else if connector != "or" {
				return fmt.Errorf("theme.css:%d: invalid media query", line)
			}
		default:
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		next++
		if next >= len(ts) || ts[next].tt != css.LeftParenthesisToken {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		inner, next, ok = extractGroup(ts, next)
		if !ok {
			return fmt.Errorf("theme.css:%d: invalid media query", line)
		}
		if err := checkMediaInParens(inner, line); err != nil {
			return err
		}
	}
	return nil
}

// checkMediaInParens validates the content inside a media condition's
// parentheses, with whitespace tokens already removed.
func checkMediaInParens(inner []cssToken, line int) error {
	if len(inner) == 0 {
		return fmt.Errorf("theme.css:%d: invalid media query", line)
	}
	if inner[0].tt == css.LeftParenthesisToken {
		return checkMediaCondition(inner, true, line)
	}
	if inner[0].tt == css.IdentToken && asciiLower(cssUnescape(inner[0].data)) == "not" {
		return checkMediaCondition(inner, true, line)
	}
	return checkMediaFeature(inner, line)
}
