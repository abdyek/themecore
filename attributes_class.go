package themecore

import (
	"errors"
	"fmt"
	"strings"
)

// maxClassLen is the largest class value in bytes.
const maxClassLen = 1024

// checkClass checks a class value: a non-empty list of tokens separated by a
// single U+0020, each of printable ASCII (U+0021 to U+007E), without
// duplicates and without the reserved prefixes umono- and compono-, compared
// case-insensitively, and at most maxClassLen bytes.
func checkClass(value string) error {
	if value == "" {
		return errors.New("must not be empty")
	}
	if len(value) > maxClassLen {
		return fmt.Errorf("longer than %d bytes", maxClassLen)
	}
	seen := map[string]bool{}
	for _, token := range strings.Split(value, " ") {
		if token == "" {
			return errors.New("tokens must be separated by a single space")
		}
		for i := 0; i < len(token); i++ {
			if token[i] < 0x21 || token[i] > 0x7e {
				return errors.New("tokens must be printable ASCII")
			}
		}
		if seen[token] {
			return fmt.Errorf("duplicate token %s", token)
		}
		seen[token] = true
		lower := strings.ToLower(token)
		if strings.HasPrefix(lower, "umono-") || strings.HasPrefix(lower, "compono-") {
			return fmt.Errorf("token %s uses a reserved prefix", token)
		}
	}
	return nil
}
