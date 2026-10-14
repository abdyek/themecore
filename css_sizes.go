package themecore

import (
	"errors"

	"github.com/tdewolff/parse/v2/css"
)

// errInvalidSizes is the error of a value that is not a valid sizes value.
var errInvalidSizes = errors.New("not a valid source size list")

// sizeFunctions are the functions a size can be written with.
var sizeFunctions = map[string]bool{"calc(": true, "min(": true, "max(": true, "clamp(": true}

// checkSizes checks a sizes value: an HTML source size list without auto.
// Entries are separated by commas; every entry but the last is a media
// condition followed by a size, and the last is a size. Media conditions use
// the media feature vocabulary and have no media type. A size is 0, a
// non-negative length in the length units of the vocabulary, or a calc(),
// min(), max() or clamp() of such lengths and numbers. Percentages are
// rejected. themecore parses the value itself.
func checkSizes(value string) error {
	toks, err := lexCSS([]byte(value))
	if err != nil {
		return errInvalidSizes
	}
	var entries [][]cssToken
	var entry []cssToken
	depth := 0
	for _, t := range toks {
		switch t.tt {
		case css.WhitespaceToken:
			continue
		case css.FunctionToken, css.LeftParenthesisToken:
			depth++
		case css.RightParenthesisToken:
			depth--
		case css.CommaToken:
			if depth == 0 {
				entries = append(entries, entry)
				entry = nil
				continue
			}
		}
		entry = append(entry, t)
	}
	entries = append(entries, entry)

	for i, e := range entries {
		if len(e) == 0 {
			return errInvalidSizes
		}
		start := sizeStart(e)
		if start < 0 || !isSize(e[start:]) {
			return errInvalidSizes
		}
		cond := e[:start]
		if i == len(entries)-1 {
			if len(cond) != 0 {
				return errInvalidSizes
			}
			continue
		}
		if len(cond) == 0 || checkMediaCondition(cond, true, 0) != nil {
			return errInvalidSizes
		}
	}
	return nil
}

// sizeStart returns the index of the first token of the size at the end of
// the entry e: its last token, or the function that the last parenthesis
// closes. It returns -1 if the entry ends with a parenthesized group.
func sizeStart(e []cssToken) int {
	last := len(e) - 1
	if e[last].tt != css.RightParenthesisToken {
		return last
	}
	depth := 0
	for i := last; i >= 0; i-- {
		switch e[i].tt {
		case css.RightParenthesisToken:
			depth++
		case css.FunctionToken, css.LeftParenthesisToken:
			depth--
			if depth == 0 {
				if e[i].tt == css.FunctionToken {
					return i
				}
				return -1
			}
		}
	}
	return -1
}

// isSize reports whether s, without whitespace, is a size.
func isSize(s []cssToken) bool {
	if len(s) == 1 {
		t := s[0]
		switch t.tt {
		case css.NumberToken:
			return string(t.data) == "0"
		case css.DimensionToken:
			return t.data[0] != '-' && lengthUnits[dimensionUnit(t)]
		}
		return false
	}
	if s[0].tt != css.FunctionToken || !sizeFunctions[asciiLower(cssUnescape(s[0].data))] {
		return false
	}
	for _, t := range s[1:] {
		switch t.tt {
		case css.NumberToken, css.CommaToken, css.LeftParenthesisToken, css.RightParenthesisToken:
		case css.DimensionToken:
			if !lengthUnits[dimensionUnit(t)] {
				return false
			}
		case css.DelimToken:
			if d := string(t.data); d != "+" && d != "-" && d != "*" && d != "/" {
				return false
			}
		case css.FunctionToken:
			if !sizeFunctions[asciiLower(cssUnescape(t.data))] {
				return false
			}
		default:
			return false
		}
	}
	return true
}
