package themecore

import (
	"bytes"
	"errors"

	"github.com/pelletier/go-toml/v2"
)

// decodeTOML decodes the TOML v1.0.0 document b into v. Keys that v does not
// define are rejected.
func decodeTOML(b []byte, v any) error {
	if err := checkEscapes(b); err != nil {
		return err
	}
	d := toml.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

// checkEscapes rejects the escape \e, which TOML v1.1.0 adds and go-toml
// v2.2.4 accepts. The scan is exact for valid documents: outside strings and
// comments, a valid document contains no quote and no #. Other errors are left
// to the decoder.
func checkEscapes(b []byte) error {
	const (
		stOutside = iota
		stComment
		stBasic
		stMultilineBasic
		stLiteral
		stMultilineLiteral
	)
	state := stOutside
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch state {
		case stOutside:
			switch c {
			case '#':
				state = stComment
			case '"':
				if i+2 < len(b) && b[i+1] == '"' && b[i+2] == '"' {
					state = stMultilineBasic
					i += 2
				} else {
					state = stBasic
				}
			case '\'':
				if i+2 < len(b) && b[i+1] == '\'' && b[i+2] == '\'' {
					state = stMultilineLiteral
					i += 2
				} else {
					state = stLiteral
				}
			}
		case stComment:
			if c == '\n' {
				state = stOutside
			}
		case stBasic:
			switch c {
			case '\\':
				if i+1 < len(b) {
					if b[i+1] == 'e' {
						return errors.New(`toml: the escape \e is not defined in TOML v1.0.0`)
					}
					i++
				}
			case '"':
				state = stOutside
			}
		case stMultilineBasic:
			switch c {
			case '\\':
				if i+1 < len(b) {
					if b[i+1] == 'e' {
						return errors.New(`toml: the escape \e is not defined in TOML v1.0.0`)
					}
					i++
				}
			case '"':
				j := i
				for j < len(b) && b[j] == '"' {
					j++
				}
				if j-i >= 3 {
					state = stOutside
					i = j - 1
				}
			}
		case stLiteral:
			if c == '\'' {
				state = stOutside
			}
		case stMultilineLiteral:
			if c == '\'' {
				j := i
				for j < len(b) && b[j] == '\'' {
					j++
				}
				if j-i >= 3 {
					state = stOutside
					i = j - 1
				}
			}
		}
	}
	return nil
}
