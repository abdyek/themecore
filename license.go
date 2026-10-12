package themecore

import (
	"errors"
	"unicode/utf8"
)

// checkLicense checks that LICENSE, when the package has one, is valid UTF-8.
// Its content is not checked otherwise: the license text is the theme
// author's responsibility.
func checkLicense(p Package) error {
	content, ok := p.Files["LICENSE"]
	if !ok {
		return nil
	}
	if !utf8.Valid(content) {
		return errors.New("package: LICENSE: not valid UTF-8")
	}
	return nil
}
