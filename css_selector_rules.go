package themecore

import (
	"fmt"
)

// hasScopePseudo reports whether c contains a pseudo-class named "scope"
// directly in its pseudos list (not nested in the selector argument of
// another pseudo).
func hasScopePseudo(c selCompound) bool {
	for _, p := range c.pseudos {
		if p.name == "scope" {
			return true
		}
	}
	return false
}

// checkSelector parses the selector list of a style rule and checks the
// selector rules (themecore, Karar 6 and 14).
func checkSelector(prelude []cssToken, line int) error {
	list, err := parseSelectorList(prelude, false, line)
	if err != nil {
		return err
	}
	for _, cx := range list {
		for i, part := range cx.parts {
			if err := checkCompound(part, line); err != nil {
				return err
			}
			if i > 0 && hasScopePseudo(part) {
				return fmt.Errorf("theme.css:%d: :scope is only allowed in the first compound selector", line)
			}
		}
		if hasScopePseudo(cx.parts[0]) {
			if len(cx.parts) > 1 && cx.combs[0] != ' ' && cx.combs[0] != '>' {
				return fmt.Errorf("theme.css:%d: the combinator after :scope must be a space or >", line)
			}
			if err := checkScopeCompound(cx.parts[0], line); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkCompound checks a compound selector and, recursively, the selector
// arguments of its pseudo-classes.
func checkCompound(c selCompound, line int) error {
	switch c.typ {
	case "html", "body", "umono-root":
		return fmt.Errorf("theme.css:%d: %s is not allowed", line, c.typ)
	}
	for _, a := range c.attrs {
		if a.name == "id" {
			return fmt.Errorf("theme.css:%d: id selectors are not allowed", line)
		}
	}
	for _, p := range c.pseudos {
		if len(p.list) == 0 {
			continue
		}
		switch p.name {
		case "is", "where", "not", "nth-child", "nth-last-child":
			for _, cx := range p.list {
				if cx.lead != 0 || len(cx.parts) != 1 {
					return fmt.Errorf("theme.css:%d: combinators are not allowed inside :%s()", line, p.name)
				}
			}
		}
		for _, cx := range p.list {
			for _, part := range cx.parts {
				if hasScopePseudo(part) {
					return fmt.Errorf("theme.css:%d: :scope is only allowed in the first compound selector", line)
				}
				if err := checkCompound(part, line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// checkScopeCompound checks the compound that contains :scope: it must not
// observe <umono-root>'s siblings (Karar 6).
func checkScopeCompound(c selCompound, line int) error {
	for _, p := range c.pseudos {
		if _, ok := siblingPseudoClasses[p.name]; ok {
			return fmt.Errorf("theme.css:%d: sibling-dependent pseudo-classes are not allowed in the :scope compound", line)
		}
		if p.name == "has" {
			for _, cx := range p.list {
				if cx.lead == '+' || cx.lead == '~' {
					return fmt.Errorf("theme.css:%d: :has() with + or ~ is not allowed in the :scope compound", line)
				}
			}
		}
		switch p.name {
		case "is", "where", "not":
			for _, cx := range p.list {
				for _, part := range cx.parts {
					if err := checkScopeCompound(part, line); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
