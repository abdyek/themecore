package themecore

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// selComplex is a complex selector: compound selectors joined by
// combinators. lead is the leading combinator of a relative selector in
// :has() ('>', '+' or '~'), 0 if there is none. combs[i] joins parts[i] and
// parts[i+1]: ' ', '>', '+' or '~'.
type selComplex struct {
	lead  byte
	parts []selCompound
	combs []byte
}

// selCompound is a compound selector. typ is the type selector with escapes
// resolved and ASCII lower-cased ("*" for the universal selector, "" when
// no type selector is present). classes lists the class selectors with
// escapes resolved. attrs lists the attribute selectors in source order.
// pseudos lists the pseudo-classes in source order. elements lists the
// pseudo-elements in source order.
type selCompound struct {
	typ      string
	classes  []string
	attrs    []selAttr
	pseudos  []selPseudo
	elements []string
}

// selAttr is an attribute selector. name is the attribute name with
// escapes resolved and ASCII lower-cased.
type selAttr struct {
	name string
}

// selPseudo is a pseudo-class. name is the pseudo-class name with escapes
// resolved and ASCII lower-cased. fn is true when the pseudo-class is
// written as a function (the name is followed by "(" and an argument
// list). list holds the parsed selector argument of :is(), :where(),
// :not() and :has(), and the parsed of-list of :nth-child() and
// :nth-last-child().
type selPseudo struct {
	name string
	fn   bool
	list []selComplex
}

// pseudoArg is the kind of argument accepted by a functional pseudo-class.
type pseudoArg int

const (
	// argCompoundList accepts a comma-separated list of compound selectors
	// (used by :is(), :where() and :not()).
	argCompoundList pseudoArg = iota
	// argRelativeList accepts a comma-separated list of relative complex
	// selectors (used by :has()).
	argRelativeList
	// argNth accepts an An+B index expression (used by :nth-of-type() and
	// :nth-last-of-type()).
	argNth
	// argNthOf accepts an An+B index expression optionally followed by the
	// "of" keyword and a comma-separated list of compound selectors
	// (used by :nth-child() and :nth-last-child()).
	argNthOf
	// argLang accepts a comma-separated list of one or more ident or
	// string tokens (used by :lang()).
	argLang
	// argDir accepts exactly one ident equal to "ltr" or "rtl" (used by
	// :dir()).
	argDir
)

// pseudoClasses is the closed list of pseudo-class names accepted in v0.9.
// A name not listed here is rejected, and adding one is a Feature row
// (css.pseudo.<name>).
var pseudoClasses = map[string]struct{}{
	"scope":         {},
	"hover":         {},
	"active":        {},
	"focus":         {},
	"focus-visible": {},
	"focus-within":  {},
	"link":          {},
	"visited":       {},
	"any-link":      {},
	"empty":         {},
	"first-child":   {},
	"last-child":    {},
	"only-child":    {},
	"first-of-type": {},
	"last-of-type":  {},
	"only-of-type":  {},
	"open":          {},
}

// pseudoFunctions maps the name of a functional pseudo-class accepted in
// v0.9 to the kind of argument it expects. A name not listed here is
// rejected, and adding one is a Feature row (css.pseudo.<name>).
var pseudoFunctions = map[string]pseudoArg{
	"is":               argCompoundList,
	"where":            argCompoundList,
	"not":              argCompoundList,
	"has":              argRelativeList,
	"nth-child":        argNthOf,
	"nth-last-child":   argNthOf,
	"nth-of-type":      argNth,
	"nth-last-of-type": argNth,
	"lang":             argLang,
	"dir":              argDir,
}

// pseudoElements is the closed list of pseudo-element names accepted in
// v0.9. A name not listed here is rejected, and adding one is a Feature
// row (css.pseudo.<name>).
var pseudoElements = map[string]struct{}{
	"before":          {},
	"after":           {},
	"marker":          {},
	"first-line":      {},
	"first-letter":    {},
	"selection":       {},
	"details-content": {},
}

// legacyPseudoElements lists the pseudo-elements that are written with a
// single colon in legacy CSS. themecore rejects them with a hint pointing
// at the two-colon form.
var legacyPseudoElements = map[string]struct{}{
	"before":       {},
	"after":        {},
	"first-line":   {},
	"first-letter": {},
}

// siblingPseudoClasses lists the pseudo-classes whose meaning depends on
// <umono-root>'s siblings in the :scope compound (Karar 6). The selector
// parser uses this set to scope sibling checks to the :scope compound.
var siblingPseudoClasses = map[string]struct{}{
	"first-child":      {},
	"last-child":       {},
	"only-child":       {},
	"nth-child":        {},
	"nth-last-child":   {},
	"first-of-type":    {},
	"last-of-type":     {},
	"only-of-type":     {},
	"nth-of-type":      {},
	"nth-last-of-type": {},
}

// anbPattern matches the textual form of an An+B index expression as
// accepted by :nth-child(), :nth-last-child(), :nth-of-type() and
// :nth-last-of-type(). The expression is either "odd", "even", an integer
// possibly signed, or an integer coefficient of "n" optionally followed by
// a signed integer step (the "B" part).
var anbPattern = regexp.MustCompile(`(?i)^(odd|even|[+-]?\d+|[+-]?\d*n(\s*[+-]\s*\d+)?)$`)

// tokensText concatenates the data of toks into a single string. A
// whitespace token contributes a single space; every other token
// contributes its raw data.
func tokensText(toks []cssToken) string {
	var sb strings.Builder
	for _, t := range toks {
		if t.tt == css.WhitespaceToken {
			sb.WriteByte(' ')
			continue
		}
		sb.Write(t.data)
	}
	return sb.String()
}

// checkAnB reports whether the textual form of the trimmed tokens of toks
// matches anbPattern. name is the name of the pseudo-class whose argument
// is being validated and is included in the error message. line is the
// line reported on validation failure.
func checkAnB(toks []cssToken, name string, line int) error {
	if !anbPattern.MatchString(tokensText(trimSpace(toks))) {
		return fmt.Errorf("theme.css:%d: invalid argument of :%s()", line, name)
	}
	return nil
}

// splitOf finds the first ident token in toks at bracket depth 0 whose
// value (after ASCII lower-casing and resolving CSS escapes) is "of" and
// returns the tokens before and after it (the "of" token itself is
// dropped). Bracket depth counts parentheses, brackets, and function
// tokens, so an "of" inside a nested selector list is not matched. If no
// such ident is found, splitOf returns toks as the first part, nil as the
// second part, and false.
func splitOf(toks []cssToken) (anb, of []cssToken, found bool) {
	depth := 0
	for i, t := range toks {
		switch t.tt {
		case css.LeftParenthesisToken, css.LeftBracketToken, css.FunctionToken:
			depth++
		case css.RightParenthesisToken, css.RightBracketToken:
			if depth > 0 {
				depth--
			}
		case css.IdentToken:
			if depth == 0 && asciiLower(cssUnescape(t.data)) == "of" {
				return toks[:i], toks[i+1:], true
			}
		}
	}
	return toks, nil, false
}

// checkLang reports whether the trimmed tokens of toks form a valid
// argument to :lang(): a non-empty comma-separated list whose every item,
// trimmed of whitespace, is exactly one ident or string token.
func checkLang(toks []cssToken, line int) error {
	items := splitComma(trimSpace(toks))
	if len(items) == 0 {
		return fmt.Errorf("theme.css:%d: invalid argument of :lang()", line)
	}
	for _, item := range items {
		item = trimSpace(item)
		if len(item) != 1 {
			return fmt.Errorf("theme.css:%d: invalid argument of :lang()", line)
		}
		t := item[0]
		if t.tt != css.IdentToken && t.tt != css.StringToken {
			return fmt.Errorf("theme.css:%d: invalid argument of :lang()", line)
		}
	}
	return nil
}

// checkDir reports whether the trimmed tokens of toks form a valid
// argument to :dir(): exactly one ident whose value (after ASCII
// lower-casing and resolving CSS escapes) is "ltr" or "rtl".
func checkDir(toks []cssToken, line int) error {
	trimmed := trimSpace(toks)
	if len(trimmed) != 1 || trimmed[0].tt != css.IdentToken {
		return fmt.Errorf("theme.css:%d: invalid argument of :dir()", line)
	}
	s := asciiLower(cssUnescape(trimmed[0].data))
	if s != "ltr" && s != "rtl" {
		return fmt.Errorf("theme.css:%d: invalid argument of :dir()", line)
	}
	return nil
}

// parsePseudo parses a pseudo-class or pseudo-element that starts with the
// colon at toks[i]. It returns either a pseudo-class or, when element is
// true, the name of a pseudo-element, and the index after it.
func parsePseudo(toks []cssToken, i int, line int) (p selPseudo, element string, next int, err error) {
	if i+1 >= len(toks) {
		return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
	}
	tok1 := toks[i+1]
	switch tok1.tt {
	case css.ColonToken:
		if i+2 >= len(toks) {
			return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
		}
		tok2 := toks[i+2]
		switch tok2.tt {
		case css.IdentToken:
			name := asciiLower(cssUnescape(tok2.data))
			if _, ok := pseudoElements[name]; ok {
				return p, name, i + 3, nil
			}
			return p, "", i, fmt.Errorf("theme.css:%d: unknown pseudo-element ::%s", line, name)
		case css.FunctionToken:
			name := asciiLower(cssUnescape(tok2.data[:len(tok2.data)-1]))
			return p, "", i, fmt.Errorf("theme.css:%d: unknown pseudo-element ::%s()", line, name)
		default:
			return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
		}
	case css.IdentToken:
		name := asciiLower(cssUnescape(tok1.data))
		if _, ok := legacyPseudoElements[name]; ok {
			return p, "", i, fmt.Errorf("theme.css:%d: use ::%s", line, name)
		}
		if name == "root" {
			return p, "", i, fmt.Errorf("theme.css:%d: :root is not allowed", line)
		}
		if name == "target" || name == "target-within" {
			return p, "", i, fmt.Errorf("theme.css:%d: id selectors are not allowed", line)
		}
		if _, ok := pseudoClasses[name]; ok {
			return selPseudo{name: name}, "", i + 2, nil
		}
		return p, "", i, fmt.Errorf("theme.css:%d: unknown pseudo-class :%s", line, name)
	case css.FunctionToken:
		name := asciiLower(cssUnescape(tok1.data[:len(tok1.data)-1]))
		depth := 1
		j := i + 2
		for j < len(toks) {
			switch toks[j].tt {
			case css.LeftParenthesisToken, css.LeftBracketToken, css.FunctionToken:
				depth++
			case css.RightParenthesisToken, css.RightBracketToken:
				depth--
			}
			if depth == 0 {
				break
			}
			j++
		}
		if depth != 0 {
			return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
		}
		arg := toks[i+2 : j]
		if name == "target" || name == "target-within" {
			return p, "", i, fmt.Errorf("theme.css:%d: id selectors are not allowed", line)
		}
		kind, ok := pseudoFunctions[name]
		if !ok {
			return p, "", i, fmt.Errorf("theme.css:%d: unknown pseudo-class :%s()", line, name)
		}
		var list []selComplex
		switch kind {
		case argCompoundList:
			if len(trimSpace(arg)) == 0 {
				return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
			}
			list, err = parseSelectorList(arg, false, line)
			if err != nil {
				return p, "", i, err
			}
		case argRelativeList:
			if len(trimSpace(arg)) == 0 {
				return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
			}
			list, err = parseSelectorList(arg, true, line)
			if err != nil {
				return p, "", i, err
			}
		case argNth:
			if err = checkAnB(arg, name, line); err != nil {
				return p, "", i, err
			}
		case argNthOf:
			anb, of, found := splitOf(arg)
			if err = checkAnB(anb, name, line); err != nil {
				return p, "", i, err
			}
			if found {
				if len(trimSpace(of)) == 0 {
					return p, "", i, fmt.Errorf("theme.css:%d: invalid argument of :%s()", line, name)
				}
				list, err = parseSelectorList(of, false, line)
				if err != nil {
					return p, "", i, err
				}
			}
		case argLang:
			if err = checkLang(arg, line); err != nil {
				return p, "", i, err
			}
		case argDir:
			if err = checkDir(arg, line); err != nil {
				return p, "", i, err
			}
		}
		return selPseudo{name: name, fn: true, list: list}, "", j + 1, nil
	default:
		return p, "", i, fmt.Errorf("theme.css:%d: invalid selector", line)
	}
}

// parseAttr parses the content between the brackets of an attribute
// selector. toks is that content already trimmed of whitespace. The first
// token must be an ident (the attribute name); any namespace separator
// delim ('|' anywhere, '*' as the first token) is rejected. After the
// name, optional whitespace is followed either by the end of toks or by a
// matcher (the delim '=', or css.IncludeMatchToken, css.DashMatchToken,
// css.PrefixMatchToken, css.SuffixMatchToken, css.SubstringMatchToken),
// optional whitespace, a value (ident or string), optional whitespace, an
// optional flag ident "i" or "s", optional whitespace, and the end.
// Anything else is rejected as "invalid attribute selector".
func parseAttr(toks []cssToken, line int) (selAttr, error) {
	if len(toks) == 0 {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	for _, t := range toks {
		if t.tt == css.DelimToken && len(t.data) > 0 && t.data[0] == '|' {
			return selAttr{}, fmt.Errorf("theme.css:%d: namespace prefixes are not allowed", line)
		}
	}
	if toks[0].tt == css.DelimToken && len(toks[0].data) > 0 && toks[0].data[0] == '*' {
		return selAttr{}, fmt.Errorf("theme.css:%d: namespace prefixes are not allowed", line)
	}
	if toks[0].tt != css.IdentToken {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	name := asciiLower(cssUnescape(toks[0].data))
	i := 1
	for i < len(toks) && toks[i].tt == css.WhitespaceToken {
		i++
	}
	if i == len(toks) {
		return selAttr{name: name}, nil
	}
	t := toks[i]
	matched := false
	switch t.tt {
	case css.IncludeMatchToken, css.DashMatchToken, css.PrefixMatchToken, css.SuffixMatchToken, css.SubstringMatchToken:
		matched = true
	case css.DelimToken:
		if len(t.data) > 0 && t.data[0] == '=' {
			matched = true
		}
	}
	if !matched {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	i++
	for i < len(toks) && toks[i].tt == css.WhitespaceToken {
		i++
	}
	if i == len(toks) {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	if toks[i].tt != css.IdentToken && toks[i].tt != css.StringToken {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	i++
	for i < len(toks) && toks[i].tt == css.WhitespaceToken {
		i++
	}
	if i < len(toks) {
		if toks[i].tt != css.IdentToken {
			return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
		}
		flag := asciiLower(cssUnescape(toks[i].data))
		if flag != "i" && flag != "s" {
			return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
		}
		i++
		for i < len(toks) && toks[i].tt == css.WhitespaceToken {
			i++
		}
	}
	if i != len(toks) {
		return selAttr{}, fmt.Errorf("theme.css:%d: invalid attribute selector", line)
	}
	return selAttr{name: name}, nil
}

// parseCompound parses the compound selector that starts at toks[i] and
// returns the index after it. It stops at whitespace, a combinator delim
// ('>', '+', '~') or the end of toks.
func parseCompound(toks []cssToken, i int, line int) (c selCompound, next int, err error) {
	if i < len(toks) {
		t := toks[i]
		if t.tt == css.IdentToken {
			c.typ = asciiLower(cssUnescape(t.data))
			i++
			if i < len(toks) && toks[i].tt == css.DelimToken && len(toks[i].data) == 1 && toks[i].data[0] == '|' {
				return c, i, fmt.Errorf("theme.css:%d: namespace prefixes are not allowed", line)
			}
		} else if t.tt == css.DelimToken && len(t.data) == 1 && t.data[0] == '*' {
			c.typ = "*"
			i++
			if i < len(toks) && toks[i].tt == css.DelimToken && len(toks[i].data) == 1 && toks[i].data[0] == '|' {
				return c, i, fmt.Errorf("theme.css:%d: namespace prefixes are not allowed", line)
			}
		}
	}
	for i < len(toks) {
		t := toks[i]
		if t.tt == css.WhitespaceToken {
			break
		}
		if t.tt == css.DelimToken && len(t.data) == 1 {
			b := t.data[0]
			if b == '>' || b == '+' || b == '~' {
				break
			}
			if b == '|' {
				return c, i, fmt.Errorf("theme.css:%d: namespace prefixes are not allowed", line)
			}
			if b == '&' {
				return c, i, fmt.Errorf("theme.css:%d: & is not allowed", line)
			}
		}
		if len(c.elements) > 0 {
			return c, i, fmt.Errorf("theme.css:%d: pseudo-elements must come last", line)
		}
		if t.tt == css.HashToken {
			return c, i, fmt.Errorf("theme.css:%d: id selectors are not allowed", line)
		}
		if t.tt == css.DelimToken && len(t.data) == 1 && t.data[0] == '.' {
			if i+1 < len(toks) && toks[i+1].tt == css.IdentToken {
				c.classes = append(c.classes, cssUnescape(toks[i+1].data))
				i += 2
				continue
			}
			return c, i, fmt.Errorf("theme.css:%d: invalid selector", line)
		}
		if t.tt == css.LeftBracketToken {
			depth := 1
			j := i + 1
			for j < len(toks) {
				if toks[j].tt == css.LeftBracketToken {
					depth++
				} else if toks[j].tt == css.RightBracketToken {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			if j >= len(toks) {
				return c, i, fmt.Errorf("theme.css:%d: invalid selector", line)
			}
			attr, aerr := parseAttr(trimSpace(toks[i+1:j]), line)
			if aerr != nil {
				return c, i, aerr
			}
			c.attrs = append(c.attrs, attr)
			i = j + 1
			continue
		}
		if t.tt == css.ColonToken {
			p, element, n, perr := parsePseudo(toks, i, line)
			if perr != nil {
				return c, i, perr
			}
			if element != "" {
				c.elements = append(c.elements, element)
			} else {
				c.pseudos = append(c.pseudos, p)
			}
			i = n
			continue
		}
		return c, i, fmt.Errorf("theme.css:%d: invalid selector", line)
	}
	if c.typ == "" && len(c.classes) == 0 && len(c.attrs) == 0 && len(c.pseudos) == 0 && len(c.elements) == 0 {
		return c, i, fmt.Errorf("theme.css:%d: invalid selector", line)
	}
	return c, i, nil
}

// splitTopComma splits toks at commas at bracket depth 0; commas inside
// "(", "[" and function tokens do not split.
func splitTopComma(toks []cssToken) [][]cssToken {
	var items [][]cssToken
	var current []cssToken
	depth := 0
	for _, t := range toks {
		switch t.tt {
		case css.LeftParenthesisToken, css.LeftBracketToken, css.FunctionToken:
			depth++
		case css.RightParenthesisToken, css.RightBracketToken:
			if depth > 0 {
				depth--
			}
		}
		if t.tt == css.CommaToken && depth == 0 {
			items = append(items, current)
			current = nil
			continue
		}
		current = append(current, t)
	}
	items = append(items, current)
	return items
}

// parseSelectorList parses a comma-separated selector list. relative allows
// each complex selector to start with a combinator, as in the argument of
// :has().
func parseSelectorList(toks []cssToken, relative bool, line int) ([]selComplex, error) {
	items := splitTopComma(toks)
	var list []selComplex
	for _, item := range items {
		item = trimSpace(item)
		if len(item) == 0 {
			return nil, fmt.Errorf("theme.css:%d: invalid selector", line)
		}
		c, err := parseComplex(item, relative, line)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// parseComplex parses a complex selector.
func parseComplex(toks []cssToken, relative bool, line int) (selComplex, error) {
	var s selComplex
	i := 0
	if i < len(toks) {
		t := toks[i]
		if t.tt == css.DelimToken && len(t.data) == 1 {
			b := t.data[0]
			if b == '>' || b == '+' || b == '~' {
				if !relative {
					return s, fmt.Errorf("theme.css:%d: a selector cannot start with a combinator", line)
				}
				s.lead = b
				i++
				for i < len(toks) && toks[i].tt == css.WhitespaceToken {
					i++
				}
			}
		}
	}
	for {
		c, next, err := parseCompound(toks, i, line)
		if err != nil {
			return s, err
		}
		s.parts = append(s.parts, c)
		if len(c.elements) > 0 {
			j := next
			for j < len(toks) && toks[j].tt == css.WhitespaceToken {
				j++
			}
			if j < len(toks) {
				return s, fmt.Errorf("theme.css:%d: pseudo-elements must come last", line)
			}
		}
		i = next
		ws := false
		for i < len(toks) && toks[i].tt == css.WhitespaceToken {
			ws = true
			i++
		}
		if i == len(toks) {
			return s, nil
		}
		t := toks[i]
		if t.tt == css.DelimToken && len(t.data) == 1 {
			b := t.data[0]
			if b == '>' || b == '+' || b == '~' {
				s.combs = append(s.combs, b)
				i++
				for i < len(toks) && toks[i].tt == css.WhitespaceToken {
					i++
				}
				if i == len(toks) {
					return s, fmt.Errorf("theme.css:%d: invalid selector", line)
				}
				continue
			}
		}
		if ws {
			s.combs = append(s.combs, ' ')
			continue
		}
		return s, fmt.Errorf("theme.css:%d: invalid selector", line)
	}
}
