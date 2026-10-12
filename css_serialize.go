package themecore

import (
	"github.com/tdewolff/parse/v2/css"
)

// serializeCSS writes rules in the canonical grammar (themecore, Karar 9).
// Every maximal run of consecutive style rules, at the top level and in
// every @media, @supports and @layer block, is wrapped in one
// "@scope (umono-root){...}" block; source order is kept. Each top-level
// item is written on its own line. The format is not part of the Theme API
// (Karar 10).
func serializeCSS(rules []cssRule) []byte {
	var buf []byte
	writeTopLevel(&buf, rules)
	return buf
}

// writeTopLevel writes rules to buf as the top level of a stylesheet.
// Consecutive style rules are grouped into a single scope block; an
// at-rule is written as one item. Every item is followed by "\n". An
// empty rule list produces an empty output, not "\n".
func writeTopLevel(buf *[]byte, rules []cssRule) {
	writeRuleList(buf, rules, true)
}

// writeBlockContent writes rules to buf inside a @media, @supports or
// @layer block. Consecutive style rules are grouped into a single scope
// block; at-rules are written inline. Items are concatenated without
// any separating newline.
func writeBlockContent(buf *[]byte, rules []cssRule) {
	writeRuleList(buf, rules, false)
}

// writeRuleList writes rules to buf, grouping consecutive style rules
// into a single scope block. After every item (a scope block or an
// at-rule), if newline is true, a "\n" is written.
func writeRuleList(buf *[]byte, rules []cssRule, newline bool) {
	i := 0
	for i < len(rules) {
		if rules[i].at == "" {
			j := i
			for j < len(rules) && rules[j].at == "" {
				j++
			}
			writeScopeBlock(buf, rules[i:j])
			if newline {
				*buf = append(*buf, '\n')
			}
			i = j
			continue
		}
		writeAtRule(buf, rules[i])
		if newline {
			*buf = append(*buf, '\n')
		}
		i++
	}
}

// writeScopeBlock writes "@scope (umono-root){" followed by the style
// rules in rules concatenated and the closing "}".
func writeScopeBlock(buf *[]byte, rules []cssRule) {
	*buf = append(*buf, "@scope (umono-root){"...)
	for _, r := range rules {
		writeStyleRule(buf, r)
	}
	*buf = append(*buf, '}')
}

// writeStyleRule writes r as prelude (compact) + "{" + declarations
// joined by ";" + "}".
func writeStyleRule(buf *[]byte, r cssRule) {
	writeTokensCompact(buf, r.prelude)
	*buf = append(*buf, '{')
	for i, d := range r.decls {
		if i > 0 {
			*buf = append(*buf, ';')
		}
		writeDecl(buf, d)
	}
	*buf = append(*buf, '}')
}

// writeAtRule writes r as "@"+at + " "+prelude (compact, only when the
// prelude has a non-whitespace token) + "{"+content+"}" if r has a
// block, otherwise ";". The content of a @media, @supports or @layer
// block is its grouped children; the content of a @keyframes block is
// its children written as style rules, not wrapped; the content of a
// @font-face block is its declarations.
func writeAtRule(buf *[]byte, r cssRule) {
	*buf = append(*buf, '@')
	*buf = append(*buf, r.at...)
	if preludeHasToken(r.prelude) {
		*buf = append(*buf, ' ')
		writeTokensCompact(buf, r.prelude)
	}
	if !r.block {
		*buf = append(*buf, ';')
		return
	}
	*buf = append(*buf, '{')
	switch r.at {
	case "media", "supports", "layer":
		writeBlockContent(buf, r.rules)
	case "keyframes":
		for _, child := range r.rules {
			writeStyleRule(buf, child)
		}
	case "font-face":
		for i, d := range r.decls {
			if i > 0 {
				*buf = append(*buf, ';')
			}
			writeDecl(buf, d)
		}
	}
	*buf = append(*buf, '}')
}

// preludeHasToken reports whether toks contains any token other than a
// whitespace token.
func preludeHasToken(toks []cssToken) bool {
	for _, t := range toks {
		if t.tt != css.WhitespaceToken {
			return true
		}
	}
	return false
}

// writeDecl writes d as rawName + ":" + value, followed by "!important"
// when d is marked important. The value of a custom property (a name
// starting with "--") keeps every whitespace as one space; every other
// value is written in compact mode.
func writeDecl(buf *[]byte, d cssDecl) {
	*buf = append(*buf, d.rawName...)
	*buf = append(*buf, ':')
	if isCustomPropertyName(d.name) {
		writeTokensSpaces(buf, d.value)
	} else {
		writeTokensCompact(buf, d.value)
	}
	if d.important {
		*buf = append(*buf, "!important"...)
	}
}

// isCustomPropertyName reports whether name starts with "--", the marker
// of a CSS custom property.
func isCustomPropertyName(name string) bool {
	return len(name) >= 2 && name[0] == '-' && name[1] == '-'
}

// writeTokensCompact writes toks to buf in compact mode. Every
// non-whitespace token is written as its data. A whitespace token is
// written as one space, except when it should be dropped: the previous
// token is a comma, a colon, "(", "[", a function token, or the delim
// ">", or the next token is a comma, ")", "]", or the delim ">".
func writeTokensCompact(buf *[]byte, toks []cssToken) {
	var prev cssToken
	hasPrev := false
	for i, t := range toks {
		if t.tt == css.WhitespaceToken {
			if shouldDropWhitespace(prev, hasPrev, toks, i) {
				continue
			}
			*buf = append(*buf, ' ')
			prev = t
			hasPrev = true
			continue
		}
		*buf = append(*buf, t.data...)
		prev = t
		hasPrev = true
	}
}

// writeTokensSpaces writes toks to buf, replacing every whitespace
// token with exactly one space and every non-whitespace token with its
// data. No whitespace is dropped.
func writeTokensSpaces(buf *[]byte, toks []cssToken) {
	for _, t := range toks {
		if t.tt == css.WhitespaceToken {
			*buf = append(*buf, ' ')
			continue
		}
		*buf = append(*buf, t.data...)
	}
}

// shouldDropWhitespace reports whether the whitespace token at index i
// in toks should be dropped in compact mode. The previous token prev is
// consulted only when hasPrev is true; otherwise only the next token is
// checked.
func shouldDropWhitespace(prev cssToken, hasPrev bool, toks []cssToken, i int) bool {
	if hasPrev {
		switch prev.tt {
		case css.CommaToken, css.ColonToken, css.LeftParenthesisToken, css.LeftBracketToken, css.FunctionToken:
			return true
		case css.DelimToken:
			if len(prev.data) > 0 && prev.data[0] == '>' {
				return true
			}
		}
	}
	if i+1 < len(toks) {
		next := toks[i+1]
		switch next.tt {
		case css.CommaToken, css.RightParenthesisToken, css.RightBracketToken:
			return true
		case css.DelimToken:
			if len(next.data) > 0 && next.data[0] == '>' {
				return true
			}
		}
	}
	return false
}
