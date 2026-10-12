package themecore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// cssToken is a token produced by the tdewolff/parse CSS lexer, annotated
// with the 1-based line number on which it starts. themecore owns the AST and
// every acceptance decision; the library only produces tokens.
type cssToken struct {
	tt   css.TokenType
	data []byte
	line int
}

// cssUnescape resolves CSS escape sequences in b according to CSS Syntax
// Level 3 ("consume an escaped code point").
//
// A backslash in b starts an escape. A backslash at the end of input yields
// U+FFFD. A backslash followed by a newline drops both bytes. A backslash
// followed by 1 to 6 hexadecimal digits yields the corresponding code point,
// with one trailing whitespace (space, tab, newline, or "\r\n" counted as
// one) consumed as well; a code point of 0, a surrogate (U+D800–U+DFFF), or
// above U+10FFFF is rendered as U+FFFD. A backslash followed by any other
// character yields that character decoded as a UTF-8 rune. Every other byte
// is copied to the result unchanged.
func cssUnescape(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for i := 0; i < len(b); {
		c := b[i]
		if c != '\\' {
			sb.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(b) {
			sb.WriteRune('\uFFFD')
			return sb.String()
		}
		c = b[i]
		if c == '\n' {
			i++
			continue
		}
		if isCSSHexDigit(c) {
			val := uint32(0)
			count := 0
			for i < len(b) && count < 6 && isCSSHexDigit(b[i]) {
				val = val*16 + cssHexValue(b[i])
				i++
				count++
			}
			if i < len(b) {
				switch b[i] {
				case ' ', '\t', '\n':
					i++
				case '\r':
					if i+1 < len(b) && b[i+1] == '\n' {
						i += 2
					}
				}
			}
			if val == 0 || (val >= 0xD800 && val <= 0xDFFF) || val > 0x10FFFF {
				sb.WriteRune('\uFFFD')
			} else {
				sb.WriteRune(rune(val))
			}
			continue
		}
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			sb.WriteByte(c)
			i++
			continue
		}
		sb.Write(b[i : i+size])
		i += size
	}
	return sb.String()
}

// isCSSHexDigit reports whether c is a hexadecimal digit (0-9, A-F, or a-f)
// as used by CSS escape sequences.
func isCSSHexDigit(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// cssHexValue returns the numeric value of c interpreted as a hexadecimal
// digit. The caller must ensure c is a valid hex digit; otherwise the result
// is 0.
func cssHexValue(c byte) uint32 {
	switch {
	case '0' <= c && c <= '9':
		return uint32(c - '0')
	case 'a' <= c && c <= 'f':
		return uint32(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return uint32(c - 'A' + 10)
	}
	return 0
}

// lexCSS lexes a CSS source buffer b and returns the resulting token stream
// with every token annotated with its 1-based starting line number.
//
// Every token's data is copied, so the returned slice does not alias the
// lexer's internal buffers and remains valid after lexCSS returns. The
// returned error (if any) has a message prefixed with "theme.css: " or
// "theme.css:<line>: ". Comment tokens are dropped; for each maximal run of
// comments, lexCSS re-lexes the concatenation of the kept tokens on either
// side and rejects the input if the run was hiding a token that would
// otherwise merge. Consecutive whitespace tokens are collapsed into the first.
func lexCSS(b []byte) ([]cssToken, error) {
	if !utf8.Valid(b) {
		return nil, errors.New("theme.css: not valid UTF-8")
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return nil, errors.New("theme.css: NUL characters are not allowed")
	}

	raw := make([]cssToken, 0, 16)
	line := 1
	l := css.NewLexer(parse.NewInputBytes(b))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			if err := l.Err(); err != io.EOF {
				return nil, fmt.Errorf("theme.css:%d: %s", line, err)
			}
			break
		}
		switch tt {
		case css.BadStringToken:
			return nil, fmt.Errorf("theme.css:%d: bad string", line)
		case css.BadURLToken:
			return nil, fmt.Errorf("theme.css:%d: bad url", line)
		case css.CDOToken, css.CDCToken:
			return nil, fmt.Errorf("theme.css:%d: <!-- and --> are not allowed", line)
		}
		cpy := make([]byte, len(data))
		copy(cpy, data)
		raw = append(raw, cssToken{tt: tt, data: cpy, line: line})
		line += countNewlines(cpy)
	}

	out := make([]cssToken, 0, len(raw))
	i := 0
	for i < len(raw) {
		if raw[i].tt != css.CommentToken {
			out = append(out, raw[i])
			i++
			continue
		}
		commentLine := raw[i].line
		for i < len(raw) && raw[i].tt == css.CommentToken {
			i++
		}
		var prev *cssToken
		if len(out) > 0 {
			prev = &out[len(out)-1]
		}
		var next *cssToken
		for j := i; j < len(raw); j++ {
			if raw[j].tt != css.CommentToken {
				next = &raw[j]
				break
			}
		}
		if prev != nil && next != nil && prev.tt != css.WhitespaceToken && next.tt != css.WhitespaceToken {
			combined := make([]byte, 0, len(prev.data)+len(next.data))
			combined = append(combined, prev.data...)
			combined = append(combined, next.data...)
			l2 := css.NewLexer(parse.NewInputBytes(combined))
			tt1, d1 := l2.Next()
			tt2, d2 := l2.Next()
			tt3, _ := l2.Next()
			if tt1 == css.ErrorToken || tt2 == css.ErrorToken || tt3 != css.ErrorToken {
				return nil, fmt.Errorf("theme.css:%d: comment separates tokens that would merge without it", commentLine)
			}
			if l2.Err() != io.EOF {
				return nil, fmt.Errorf("theme.css:%d: comment separates tokens that would merge without it", commentLine)
			}
			if tt1 != prev.tt || !bytes.Equal(d1, prev.data) || tt2 != next.tt || !bytes.Equal(d2, next.data) {
				return nil, fmt.Errorf("theme.css:%d: comment separates tokens that would merge without it", commentLine)
			}
		}
	}

	collapsed := make([]cssToken, 0, len(out))
	for _, t := range out {
		if len(collapsed) > 0 && collapsed[len(collapsed)-1].tt == css.WhitespaceToken && t.tt == css.WhitespaceToken {
			continue
		}
		collapsed = append(collapsed, t)
	}
	return collapsed, nil
}

// countNewlines counts the number of newlines in b, where "\n", "\f", and
// a "\r" not followed by "\n" each count as one.
func countNewlines(b []byte) int {
	n := 0
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\n', '\f':
			n++
		case '\r':
			if i+1 < len(b) && b[i+1] == '\n' {
				continue
			}
			n++
		}
	}
	return n
}

// cssHasComment reports whether b contains a CSS comment token.
func cssHasComment(b []byte) bool {
	l := css.NewLexer(parse.NewInputBytes(b))
	for {
		tt, _ := l.Next()
		if tt == css.ErrorToken {
			return false
		}
		if tt == css.CommentToken {
			return true
		}
	}
}
