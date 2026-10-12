package themecore

import (
	"fmt"

	"github.com/tdewolff/parse/v2/css"
)

// checkSupports checks an @supports prelude (themecore, Karar 16). Property
// and value conditions only ask what the browser supports and are not
// checked; the selector inside selector() is subject to the selector rules
// and the dictionary.
func checkSupports(toks []cssToken, line int) error {
	ts := trimSpace(toks)
	if len(ts) == 0 {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	next, err := checkSupportsCondition(ts, 0, line)
	if err != nil {
		return err
	}
	if next != len(ts) {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	return nil
}

// checkSupportsCondition parses a single @supports condition starting at
// ts[start] and returns the index of the first token after the condition.
// Whitespace tokens between parts are skipped; whitespace inside groups is
// preserved for the inner-content check.
func checkSupportsCondition(ts []cssToken, start int, line int) (int, error) {
	i := start
	for i < len(ts) && ts[i].tt == css.WhitespaceToken {
		i++
	}
	if i >= len(ts) {
		return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	if ts[i].tt == css.IdentToken && asciiLower(cssUnescape(ts[i].data)) == "not" {
		i++
		for i < len(ts) && ts[i].tt == css.WhitespaceToken {
			i++
		}
		if i >= len(ts) {
			return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		return checkSupportsGroup(ts, i, line)
	}
	next, err := checkSupportsGroup(ts, i, line)
	if err != nil {
		return next, err
	}
	i = next
	connector := ""
	for i < len(ts) {
		for i < len(ts) && ts[i].tt == css.WhitespaceToken {
			i++
		}
		if i >= len(ts) {
			break
		}
		if ts[i].tt != css.IdentToken {
			return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		kw := asciiLower(cssUnescape(ts[i].data))
		i++
		switch kw {
		case "and":
			if connector != "" && connector != "and" {
				return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
			}
			connector = "and"
		case "or":
			if connector != "" && connector != "or" {
				return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
			}
			connector = "or"
		default:
			return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		for i < len(ts) && ts[i].tt == css.WhitespaceToken {
			i++
		}
		if i >= len(ts) {
			return i, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		next, err := checkSupportsGroup(ts, i, line)
		if err != nil {
			return next, err
		}
		i = next
	}
	return i, nil
}

// checkSupportsGroup parses one group starting at ts[start] (either "(" or
// a FunctionToken) and returns the index of the first token after its
// closing ")". The only allowed function is selector(); any other function
// name or a missing ")" is rejected.
func checkSupportsGroup(ts []cssToken, start int, line int) (int, error) {
	t := ts[start]
	if t.tt != css.LeftParenthesisToken && t.tt != css.FunctionToken {
		return start, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	if t.tt == css.FunctionToken {
		name := asciiLower(cssUnescape(t.data))
		if len(name) > 0 && name[len(name)-1] == '(' {
			name = name[:len(name)-1]
		}
		if name != "selector" {
			return start, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
	}
	inner, next, ok := extractGroup(ts, start)
	if !ok {
		return start, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	if t.tt == css.FunctionToken {
		content := trimSpace(inner)
		items := splitTopComma(content)
		if len(items) != 1 {
			return next, fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		return next, checkSelector(content, line)
	}
	return next, checkSupportsParensContent(inner, line)
}

// checkSupportsParensContent validates the inner tokens of a "(" group:
// either a nested condition (starts with "(" or the keyword "not") or a
// declaration (ident, optional whitespace, colon, non-empty value).
func checkSupportsParensContent(inner []cssToken, line int) error {
	trimmed := trimSpace(inner)
	if len(trimmed) == 0 {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	if trimmed[0].tt == css.LeftParenthesisToken {
		next, err := checkSupportsCondition(trimmed, 0, line)
		if err != nil {
			return err
		}
		if next != len(trimmed) {
			return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		return nil
	}
	if trimmed[0].tt == css.IdentToken && asciiLower(cssUnescape(trimmed[0].data)) == "not" {
		next, err := checkSupportsCondition(trimmed, 0, line)
		if err != nil {
			return err
		}
		if next != len(trimmed) {
			return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
		}
		return nil
	}
	if trimmed[0].tt != css.IdentToken && trimmed[0].tt != css.CustomPropertyNameToken {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	i := 1
	for i < len(trimmed) && trimmed[i].tt == css.WhitespaceToken {
		i++
	}
	if i >= len(trimmed) || trimmed[i].tt != css.ColonToken {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	if len(trimSpace(trimmed[i+1:])) == 0 {
		return fmt.Errorf("theme.css:%d: invalid @supports condition", line)
	}
	return nil
}
