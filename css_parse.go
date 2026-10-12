package themecore

import (
	"fmt"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// cssRule is a style rule or an at-rule. For a style rule at is empty and
// prelude holds the selector. For an at-rule at is the at-rule name without
// the leading "@" (escapes resolved, ASCII lower case) and prelude holds the
// at-rule's prelude. block reports whether the rule was followed by a {}
// block. rules holds the rule list inside an at-rule block of an at-rule in
// ruleListAtRules; decls holds the declarations inside a style rule or an
// at-rule block of an at-rule in declListAtRules.
type cssRule struct {
	at      string
	prelude []cssToken
	block   bool
	rules   []cssRule
	decls   []cssDecl
}

// cssDecl is a declaration inside a style rule or a declaration-list
// at-rule. name is the unescaped property name (ASCII lower case unless the
// property is a custom property starting with "--"). value is the
// declaration value without leading or trailing whitespace and without
// "!important". important is true when the declaration carried an
// "!important" marker.
type cssDecl struct {
	name      string
	value     []cssToken
	important bool
}

// ruleListAtRules is the set of at-rule names whose block holds a rule
// list. At-rules in this set may be nested inside other rule-list
// at-rules.
var ruleListAtRules = map[string]struct{}{
	"media":     {},
	"supports":  {},
	"layer":     {},
	"scope":     {},
	"keyframes": {},
}

// declListAtRules is the set of at-rule names whose block holds a
// declaration list. At-rules in this set may not contain nested rules.
var declListAtRules = map[string]struct{}{
	"font-face": {},
}

// asciiLower lower-cases only the ASCII letters A-Z in s. Bytes outside
// A-Z, including UTF-8 continuation bytes, are copied unchanged.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// collectTokens collects tokens from toks starting at index i until a
// semicolon, left brace, or right brace is seen at bracket depth 0, or the
// end of toks is reached. A left parenthesis and a function token each push
// an expected closing right parenthesis on a stack; a left bracket pushes an
// expected closing right bracket. A right parenthesis or right bracket that
// does not match the top of the stack returns an "unbalanced brackets"
// error. A semicolon, left brace, or right brace seen while the stack is
// non-empty also returns an "unbalanced brackets" error. The returned slice
// is not trimmed of leading or trailing whitespace; next is the index of the
// token that stopped collection (or len(toks) at the end of input).
func collectTokens(toks []cssToken, i int) (out []cssToken, next int, err error) {
	stack := make([]byte, 0, 4)
	for j := i; j < len(toks); j++ {
		t := toks[j]
		switch t.tt {
		case css.LeftParenthesisToken, css.FunctionToken:
			stack = append(stack, ')')
		case css.LeftBracketToken:
			stack = append(stack, ']')
		case css.RightParenthesisToken:
			if len(stack) == 0 || stack[len(stack)-1] != ')' {
				return nil, j, fmt.Errorf("theme.css:%d: unbalanced brackets", t.line)
			}
			stack = stack[:len(stack)-1]
		case css.RightBracketToken:
			if len(stack) == 0 || stack[len(stack)-1] != ']' {
				return nil, j, fmt.Errorf("theme.css:%d: unbalanced brackets", t.line)
			}
			stack = stack[:len(stack)-1]
		case css.SemicolonToken, css.LeftBraceToken, css.RightBraceToken:
			if len(stack) > 0 {
				return nil, j, fmt.Errorf("theme.css:%d: unbalanced brackets", t.line)
			}
			return toks[i:j], j, nil
		}
	}
	return toks[i:], len(toks), nil
}

// trimSpace returns toks without leading or trailing whitespace tokens. The
// middle of the slice is preserved unchanged.
func trimSpace(toks []cssToken) []cssToken {
	start := 0
	for start < len(toks) && toks[start].tt == css.WhitespaceToken {
		start++
	}
	end := len(toks)
	for end > start && toks[end-1].tt == css.WhitespaceToken {
		end--
	}
	return toks[start:end]
}

// parseDecl builds a cssDecl from the tokens collected for it (not empty,
// already checked for blocks and brackets).
//
// The first token must be an ident or a custom-property-name token,
// otherwise parseDecl returns "expected a property name". After optional
// whitespace, the next token must be a colon, otherwise parseDecl returns
// "expected : after property name". The remaining tokens form the value,
// which is trimmed of whitespace.
//
// If the value ends with a delim "!" followed by optional whitespace and an
// ident whose asciiLower(cssUnescape) is "important", the marker is removed
// from the value, the value is trimmed again, and important is set to true.
//
// The unescaped property name is taken as-is if it starts with "--" (custom
// property) and ASCII lower-cased otherwise. A non-custom property with an
// empty value triggers "empty value"; a custom property may be empty.
func parseDecl(toks []cssToken) (cssDecl, error) {
	if toks[0].tt != css.IdentToken && toks[0].tt != css.CustomPropertyNameToken {
		return cssDecl{}, fmt.Errorf("theme.css:%d: expected a property name", toks[0].line)
	}
	name := cssUnescape(toks[0].data)
	customProperty := strings.HasPrefix(name, "--")
	if !customProperty {
		name = asciiLower(name)
	}

	j := 1
	if j < len(toks) && toks[j].tt == css.WhitespaceToken {
		j++
	}
	if j >= len(toks) || toks[j].tt != css.ColonToken {
		return cssDecl{}, fmt.Errorf("theme.css:%d: expected : after property name", toks[0].line)
	}
	colonLine := toks[j].line
	j++

	value := trimSpace(toks[j:])

	important := false
	if len(value) > 0 && value[len(value)-1].tt == css.IdentToken {
		last := value[len(value)-1]
		if asciiLower(cssUnescape(last.data)) == "important" {
			rest := value[:len(value)-1]
			rest = trimSpace(rest)
			if len(rest) > 0 && rest[len(rest)-1].tt == css.DelimToken && rest[len(rest)-1].data[0] == '!' {
				value = trimSpace(rest[:len(rest)-1])
				important = true
			}
		}
	}

	if len(value) == 0 && !customProperty {
		return cssDecl{}, fmt.Errorf("theme.css:%d: empty value", colonLine)
	}

	return cssDecl{
		name:      name,
		value:     value,
		important: important,
	}, nil
}

// cssParser walks the tokens of a stylesheet. cssParser holds the token
// stream being parsed and the index of the next token to read. Every
// method on cssParser advances p.i as it consumes tokens, and never
// rewinds it.
type cssParser struct {
	toks []cssToken
	i    int
}

// parseCSS lexes b and parses it as a top-level CSS rule list. The parser
// is strict: it never relies on error recovery, and anything a browser
// would recover from is reported as an error.
func parseCSS(b []byte) ([]cssRule, error) {
	toks, err := lexCSS(b)
	if err != nil {
		return nil, err
	}
	rules, err := (&cssParser{toks: toks}).parseRuleList(0, true)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// parseRuleList parses a rule list at the current parser position. At
// top level (topLevel is true) the function returns when it reaches the
// end of the token stream, and any "}" token is an "unexpected }" error.
// Inside a block that opened at line blockLine, the function returns
// when it reaches the end of the token stream ("unclosed block" with
// blockLine) or the closing "}" token, which is consumed.
//
// Whitespace tokens are skipped. An at-keyword token starts an at-rule;
// any other token starts a style rule.
func (p *cssParser) parseRuleList(blockLine int, topLevel bool) ([]cssRule, error) {
	var rules []cssRule
	for {
		for p.i < len(p.toks) && p.toks[p.i].tt == css.WhitespaceToken {
			p.i++
		}
		if p.i >= len(p.toks) {
			if topLevel {
				return rules, nil
			}
			return nil, fmt.Errorf("theme.css:%d: unclosed block", blockLine)
		}
		t := p.toks[p.i]
		if t.tt == css.RightBraceToken {
			if topLevel {
				return nil, fmt.Errorf("theme.css:%d: unexpected }", t.line)
			}
			p.i++
			return rules, nil
		}
		var (
			r   cssRule
			err error
		)
		if t.tt == css.AtKeywordToken {
			r, err = p.parseAtRule()
		} else {
			r, err = p.parseStyleRule()
		}
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
}

// parseAtRule parses an at-rule at the current parser position, which
// must point to an at-keyword token. The returned rule's at field is the
// at-keyword name with the leading "@" stripped, escapes resolved, and
// ASCII lower-cased. If the name is in neither ruleListAtRules nor
// declListAtRules, parseAtRule returns an "@<name> is not allowed" error
// using the line of the at-keyword.
//
// After the prelude is collected and trimmed, the next token must be ";"
// (statement form, no block) or "{" (block form). For a block the inner
// rule list (ruleListAtRules) or declaration list (declListAtRules) is
// parsed, and braceLine is the line of the opening "{". A "}" token or
// end of input is an "expected { or ;" error using the line of the
// at-keyword.
func (p *cssParser) parseAtRule() (cssRule, error) {
	nameTok := p.toks[p.i]
	name := asciiLower(cssUnescape(nameTok.data[1:]))
	p.i++
	_, inRules := ruleListAtRules[name]
	_, inDecls := declListAtRules[name]
	if !inRules && !inDecls {
		return cssRule{}, fmt.Errorf("theme.css:%d: @%s is not allowed", nameTok.line, name)
	}
	prelude, next, err := collectTokens(p.toks, p.i)
	if err != nil {
		return cssRule{}, err
	}
	p.i = next
	prelude = trimSpace(prelude)

	if p.i >= len(p.toks) || p.toks[p.i].tt == css.RightBraceToken {
		return cssRule{}, fmt.Errorf("theme.css:%d: expected { or ;", nameTok.line)
	}
	stopTok := p.toks[p.i]
	if stopTok.tt == css.SemicolonToken {
		p.i++
		return cssRule{at: name, prelude: prelude, block: false}, nil
	}
	braceLine := stopTok.line
	p.i++
	if inRules {
		rules, err := p.parseRuleList(braceLine, false)
		if err != nil {
			return cssRule{}, err
		}
		return cssRule{at: name, prelude: prelude, block: true, rules: rules}, nil
	}
	decls, err := p.parseDeclList(braceLine)
	if err != nil {
		return cssRule{}, err
	}
	return cssRule{at: name, prelude: prelude, block: true, decls: decls}, nil
}

// parseStyleRule parses a style rule at the current parser position.
// After the prelude is collected and trimmed, the next token must be
// "{". A stopping token of ";", "}", or end of input is an "expected {"
// error (at end of input, the line is taken from the last token). An
// empty prelude is an "empty prelude" error.
//
// On success, the "{" is consumed and a declaration list is parsed into
// the rule's decls field.
func (p *cssParser) parseStyleRule() (cssRule, error) {
	prelude, next, err := collectTokens(p.toks, p.i)
	if err != nil {
		return cssRule{}, err
	}
	p.i = next
	prelude = trimSpace(prelude)

	if p.i >= len(p.toks) {
		return cssRule{}, fmt.Errorf("theme.css:%d: expected {", p.toks[len(p.toks)-1].line)
	}
	stopTok := p.toks[p.i]
	if stopTok.tt == css.SemicolonToken || stopTok.tt == css.RightBraceToken {
		return cssRule{}, fmt.Errorf("theme.css:%d: expected {", stopTok.line)
	}
	if len(prelude) == 0 {
		return cssRule{}, fmt.Errorf("theme.css:%d: empty prelude", stopTok.line)
	}
	braceLine := stopTok.line
	p.i++
	decls, err := p.parseDeclList(braceLine)
	if err != nil {
		return cssRule{}, err
	}
	return cssRule{prelude: prelude, block: true, decls: decls}, nil
}

// parseDeclList parses a declaration list at the current parser
// position, which must be inside a block that opened at line blockLine.
// Leading whitespace and ";" tokens are skipped. The closing "}" is
// consumed and the function returns. End of input before the closing "}"
// is an "unclosed block" error using blockLine.
//
// For each declaration the tokens are collected with collectTokens. If a
// "{" stopped the collection, parseDeclList returns an "a block is not
// allowed here" error using the line of the "{". If end of input stopped
// it, parseDeclList returns an "unclosed block" error using blockLine.
// Otherwise the declaration is parsed with parseDecl.
func (p *cssParser) parseDeclList(blockLine int) ([]cssDecl, error) {
	var decls []cssDecl
	for {
		for p.i < len(p.toks) {
			tt := p.toks[p.i].tt
			if tt == css.WhitespaceToken || tt == css.SemicolonToken {
				p.i++
				continue
			}
			break
		}
		if p.i >= len(p.toks) {
			return nil, fmt.Errorf("theme.css:%d: unclosed block", blockLine)
		}
		if p.toks[p.i].tt == css.RightBraceToken {
			p.i++
			return decls, nil
		}
		toks, next, err := collectTokens(p.toks, p.i)
		if err != nil {
			return nil, err
		}
		if next < len(p.toks) && p.toks[next].tt == css.LeftBraceToken {
			return nil, fmt.Errorf("theme.css:%d: a block is not allowed here", p.toks[next].line)
		}
		if next >= len(p.toks) {
			return nil, fmt.Errorf("theme.css:%d: unclosed block", blockLine)
		}
		p.i = next
		d, err := parseDecl(toks)
		if err != nil {
			return nil, err
		}
		decls = append(decls, d)
	}
}
