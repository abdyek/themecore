package themecore

import (
	"fmt"
	"slices"

	"github.com/umono-cms/compono"
)

// BuiltinSet returns an error if a built-in or Markdown element open to
// themes is missing from the Compono this binary is built with: a removal
// without a Break row. Compono's extra names are allowed; they stay closed
// to themes until their row is added. Go's version selection can give a
// binary a newer Compono than the one themecore is tested with, so Umono
// and CLI call it in their tests.
func BuiltinSet() error {
	if err := checkBuiltinSet("built-in", builtins(), compono.Builtins()); err != nil {
		return err
	}
	return checkBuiltinSet("markdown element", markdownElements(), compono.MarkdownElements())
}

// checkBuiltinSet checks that every name of set is in componoNames. It
// returns an error for the first name of set that Compono does not have.
func checkBuiltinSet(kind string, set, componoNames []string) error {
	if len(set) == 0 {
		return fmt.Errorf("themecore: the %s set is empty", kind)
	}
	for _, name := range set {
		if !slices.Contains(componoNames, name) {
			return fmt.Errorf("themecore: %s %s is in themecore's set but not in Compono: a removal without a Break row", kind, name)
		}
	}
	return nil
}
