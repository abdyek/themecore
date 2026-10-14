package themecore

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// decodeTOML decodes the TOML v1.0.0 document b into v. Keys that v does not
// define are rejected with an "unknown key" error naming the first one.
func decodeTOML(b []byte, v any) error {
	if err := checkTOMLSyntax(b); err != nil {
		return err
	}
	d := toml.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	err := d.Decode(v)
	if err == nil {
		return nil
	}
	var e *toml.StrictMissingError
	if errors.As(err, &e) && len(e.Errors) > 0 {
		return fmt.Errorf("unknown key %s", strings.Join([]string(e.Errors[0].Key()), "."))
	}
	return err
}

// backslash is the escape byte of TOML basic strings.
const backslash = byte(92)

// doubleQuote and singleQuote open basic and literal strings.
const (
	doubleQuote = byte(34)
	singleQuote = byte(39)
)

// checkTOMLSyntax rejects the syntax that TOML v1.1.0 adds to TOML v1.0.0: a
// newline or a trailing comma in an inline table, the escapes e and xHH, and
// a time without seconds. go-toml v2.2.4 accepts some of them, and a newer
// go-toml chosen by another dependency of the binary could accept all of
// them. The scan is exact for valid documents: outside strings and comments,
// a valid TOML v1.0.0 document has no quote, no # and a colon only in times
// and time zone offsets. Other errors are left to the decoder.
func checkTOMLSyntax(b []byte) error {
	var open []byte
	var last byte
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '#':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			i--
			continue
		case doubleQuote, singleQuote:
			end, err := skipTOMLString(b, i)
			if err != nil {
				return err
			}
			i = end - 1
			last = c
			continue
		case '\n':
			if len(open) > 0 && open[len(open)-1] == '{' {
				return errors.New("toml: an inline table cannot span lines in TOML v1.0.0")
			}
		case '{', '[':
			open = append(open, c)
		case '}', ']':
			if c == '}' && last == ',' {
				return errors.New("toml: an inline table cannot end with a comma in TOML v1.0.0")
			}
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
		case ':':
			if err := checkTOMLTime(b, i); err != nil {
				return err
			}
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			last = c
		}
	}
	return nil
}

// skipTOMLString returns the offset right after the string that starts at
// start, and an error for an escape that TOML v1.0.0 does not define. An
// unterminated string ends at the end of b.
func skipTOMLString(b []byte, start int) (int, error) {
	quote := b[start]
	n := 1
	if start+2 < len(b) && b[start+1] == quote && b[start+2] == quote {
		n = 3
	}
	for i := start + n; i < len(b); i++ {
		c := b[i]
		if c == backslash && quote == doubleQuote {
			if i+1 < len(b) && (b[i+1] == 'e' || b[i+1] == 'x') {
				return 0, fmt.Errorf("toml: the escape %c%c is not defined in TOML v1.0.0", backslash, b[i+1])
			}
			i++
			continue
		}
		if c != quote {
			continue
		}
		if n == 1 {
			return i + 1, nil
		}
		j := i
		for j < len(b) && b[j] == quote {
			j++
		}
		if j-i >= 3 {
			return j, nil
		}
		i = j - 1
	}
	return len(b), nil
}

// checkTOMLTime rejects a time without seconds whose first colon is at i.
// The colon of seconds and of a time zone offset is skipped.
func checkTOMLTime(b []byte, i int) error {
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }
	if i < 2 || !isDigit(b[i-1]) || !isDigit(b[i-2]) {
		return nil
	}
	if i >= 3 && (b[i-3] == ':' || b[i-3] == '+' || b[i-3] == '-') {
		return nil
	}
	if i+2 < len(b) && isDigit(b[i+1]) && isDigit(b[i+2]) && (i+3 >= len(b) || b[i+3] != ':') {
		return errors.New("toml: a time must have seconds in TOML v1.0.0")
	}
	return nil
}
