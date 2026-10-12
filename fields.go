package themecore

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxURLLen is the length limit of a URL in bytes.
const maxURLLen = 2048

// maxNameLen is the length limit of a name in runes.
const maxNameLen = 64

// maxDescriptionLen is the length limit of a description in runes.
const maxDescriptionLen = 280

// idPattern is org/id, both parts kebab-case.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*/[a-z0-9]+(-[a-z0-9]+)*$`)

// contractRefPattern is a contract reference: org/id@vMAJOR.MINOR.
var contractRefPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*/[a-z0-9]+(-[a-z0-9]+)*@v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// versionPattern is a theme version: MAJOR.MINOR.PATCH without prefix,
// pre-release or build metadata.
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// contractVersionPattern is a contract version: vMAJOR.MINOR.
var contractVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// kebabPattern is a kebab-case name, such as a layout id.
var kebabPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// componentNamePattern is a component name in SCREAMING_SNAKE_CASE, the same grammar as Compono's.
var componentNamePattern = regexp.MustCompile(`^[A-Z0-9]+(?:_[A-Z0-9]+)*$`)

// checkText applies the general rule for text fields: single line,
// no control characters, no leading or trailing whitespace.
func checkText(file, key, value string) error {
	for _, r := range value {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("%s: %s: contains a control character", file, key)
		}
	}
	runes := []rune(value)
	if len(runes) > 0 {
		if unicode.IsSpace(runes[0]) || unicode.IsSpace(runes[len(runes)-1]) {
			return fmt.Errorf("%s: %s: has leading or trailing whitespace", file, key)
		}
	}
	return nil
}

// checkMaxRunes reports whether value has more than n runes.
func checkMaxRunes(file, key, value string, n int) error {
	if utf8.RuneCountInString(value) > n {
		return fmt.Errorf("%s: %s: longer than %d characters", file, key, n)
	}
	return nil
}

// checkHTTPSURL applies the URL rule: a single-line value within the URL
// length limit and an absolute https URL with a host.
func checkHTTPSURL(file, key, value string) error {
	if err := checkText(file, key, value); err != nil {
		return err
	}
	if len(value) > maxURLLen {
		return fmt.Errorf("%s: %s: longer than %d bytes", file, key, maxURLLen)
	}
	if !strings.HasPrefix(value, "https://") {
		return fmt.Errorf("%s: %s: must be an absolute https URL", file, key)
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%s: %s: must be an absolute https URL", file, key)
	}
	return nil
}

// checkSource checks a Compono source, the only multi-line text field: newlines, carriage returns and tabs are allowed, and so is leading or trailing whitespace.
func checkSource(file, key, value string) error {
	for _, r := range value {
		if (r < 0x20 && r != '\n' && r != '\r' && r != '\t') || r == 0x7F {
			return fmt.Errorf("%s: %s: contains a control character", file, key)
		}
	}
	if strings.Trim(value, " \t\r\n") == "" {
		return fmt.Errorf("%s: %s: must not be empty", file, key)
	}
	return nil
}
