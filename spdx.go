package themecore

import (
	"regexp"
	"strings"
)

var (
	spdxLicenseIDPattern  = regexp.MustCompile(`^[A-Za-z0-9.-]+\+?$`)
	spdxLicenseRefPattern = regexp.MustCompile(`^(DocumentRef-[A-Za-z0-9.-]+:)?LicenseRef-[A-Za-z0-9.-]+$`)
	spdxExceptionPattern  = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)
)

type spdxTokenKind int

const (
	spdxTokNone spdxTokenKind = iota
	spdxTokWord
	spdxTokLParen
	spdxTokRParen
)

type spdxToken struct {
	kind spdxTokenKind
	text string
}

func spdxIsWordChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '.' || c == '-' || c == ':' || c == '+'
}

// tokenizeSPDX splits expr into SPDX tokens; ok reports whether expr is
// a syntactically valid token sequence.
func tokenizeSPDX(expr string) ([]spdxToken, bool) {
	var tokens []spdxToken
	i := 0
	n := len(expr)
	if n == 0 {
		return nil, false
	}
	lastKind := spdxTokNone
	for {
		if i >= n {
			return nil, false
		}
		c := expr[i]
		var tok spdxToken
		switch {
		case c == '(':
			tok = spdxToken{kind: spdxTokLParen}
			i++
		case c == ')':
			tok = spdxToken{kind: spdxTokRParen}
			i++
		case spdxIsWordChar(c):
			start := i
			for i < n && spdxIsWordChar(expr[i]) {
				i++
			}
			tok = spdxToken{kind: spdxTokWord, text: expr[start:i]}
		default:
			return nil, false
		}
		tokens = append(tokens, tok)
		lastKind = tok.kind

		if i >= n {
			break
		}

		if expr[i] == ' ' {
			i++
			if i >= n {
				return nil, false
			}
			if expr[i] == ' ' {
				return nil, false
			}
		} else {
			if lastKind == spdxTokWord && expr[i] != '(' && expr[i] != ')' {
				return nil, false
			}
		}
	}
	return tokens, true
}

// spdxParser is a recursive descent parser over a slice of SPDX tokens.
type spdxParser struct {
	tokens []spdxToken
	pos    int
}

func (p *spdxParser) peek() (spdxToken, bool) {
	if p.pos >= len(p.tokens) {
		return spdxToken{}, false
	}
	return p.tokens[p.pos], true
}

func (p *spdxParser) consume() (spdxToken, bool) {
	if p.pos >= len(p.tokens) {
		return spdxToken{}, false
	}
	t := p.tokens[p.pos]
	p.pos++
	return t, true
}

func (p *spdxParser) expression() bool {
	if !p.andExpression() {
		return false
	}
	for {
		t, ok := p.peek()
		if !ok || t.kind != spdxTokWord || t.text != "OR" {
			return true
		}
		p.consume()
		if !p.andExpression() {
			return false
		}
	}
}

func (p *spdxParser) andExpression() bool {
	if !p.term() {
		return false
	}
	for {
		t, ok := p.peek()
		if !ok || t.kind != spdxTokWord || t.text != "AND" {
			return true
		}
		p.consume()
		if !p.term() {
			return false
		}
	}
}

func (p *spdxParser) term() bool {
	t, ok := p.peek()
	if !ok {
		return false
	}
	if t.kind == spdxTokLParen {
		p.consume()
		if !p.expression() {
			return false
		}
		t2, ok := p.peek()
		if !ok || t2.kind != spdxTokRParen {
			return false
		}
		p.consume()
		return true
	}
	if !p.simple() {
		return false
	}
	if t, ok := p.peek(); ok && t.kind == spdxTokWord && t.text == "WITH" {
		p.consume()
		if !p.exception() {
			return false
		}
	}
	return true
}

func (p *spdxParser) simple() bool {
	t, ok := p.peek()
	if !ok || t.kind != spdxTokWord {
		return false
	}
	if t.text == "AND" || t.text == "OR" || t.text == "WITH" {
		return false
	}
	if strings.HasPrefix(t.text, "LicenseRef-") || strings.HasPrefix(t.text, "DocumentRef-") {
		if !spdxLicenseRefPattern.MatchString(t.text) {
			return false
		}
	} else {
		if !spdxLicenseIDPattern.MatchString(t.text) {
			return false
		}
	}
	p.consume()
	return true
}

func (p *spdxParser) exception() bool {
	t, ok := p.peek()
	if !ok || t.kind != spdxTokWord {
		return false
	}
	if t.text == "AND" || t.text == "OR" || t.text == "WITH" {
		return false
	}
	if !spdxExceptionPattern.MatchString(t.text) {
		return false
	}
	p.consume()
	return true
}

// validSPDX reports whether expr is a valid SPDX license expression. Only
// the syntax is checked; identifiers are not looked up in the SPDX
// license list.
func validSPDX(expr string) bool {
	tokens, ok := tokenizeSPDX(expr)
	if !ok {
		return false
	}
	p := &spdxParser{tokens: tokens}
	if !p.expression() {
		return false
	}
	return p.pos == len(p.tokens)
}
