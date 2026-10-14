package themecore

import (
	"fmt"
	"strings"
)

// Warning is a finding of Lint. Where is a selector or an address, for
// example ".layout-post h1" or "post-list MAIN_MENU LINK".
type Warning struct {
	Message string
	Where   string
}

// Lint returns warnings for mistakes that are valid CSS but never match.
// Only development tools call it; it is not part of the Theme API, and
// install and activation do not call it. It reports root classes used
// without :scope and addresses whose component the layout never renders
// (themecore, Kök Sınıfı Uyarısı and Ölü Adres Uyarısı). A package whose
// attributes.toml, contract.toml or assets/theme.css cannot be read gets
// no warnings; those errors are reported by the checks.
func Lint(p Package) []Warning {
	b, ok := p.Files["attributes.toml"]
	if !ok {
		return nil
	}
	addrs, err := decodeAddresses(b)
	if err != nil {
		return nil
	}

	warnings := append(rootClassWarnings(p, addrs), deadAddressWarnings(p, addrs)...)
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}

// rootClassWarnings returns the warnings for root classes used without
// :scope in the canonical CSS of p, or nil.
func rootClassWarnings(p Package, addrs map[string]map[string]string) []Warning {
	rootClasses := make(map[string]struct{})
	for address, attrs := range addrs {
		if addressParts(address) != 1 {
			continue
		}
		class, has := attrs["class"]
		if !has {
			continue
		}
		for _, c := range strings.Fields(class) {
			rootClasses[c] = struct{}{}
		}
	}
	for address, attrs := range addrs {
		if addressParts(address) <= 1 {
			continue
		}
		class, has := attrs["class"]
		if !has {
			continue
		}
		for _, c := range strings.Fields(class) {
			delete(rootClasses, c)
		}
	}

	css, ok := p.Files[themeCSS]
	if !ok {
		return nil
	}
	rules, err := parseCSS(css)
	if err != nil {
		return nil
	}

	var warnings []Warning
	var walk func(rs []cssRule, inKeyframes bool)
	walk = func(rs []cssRule, inKeyframes bool) {
		for _, r := range rs {
			if r.at == "" {
				if inKeyframes {
					continue
				}
				for _, item := range splitTopComma(r.prelude) {
					text := tokensText(trimSpace(item))
					list, err := parseSelectorList(trimSpace(item), false, r.line)
					if err != nil {
						continue
					}
					for _, c := range list {
						found := false
						for _, part := range c.parts {
							if hasScopePseudo(part) {
								continue
							}
							for _, class := range part.classes {
								if _, isRoot := rootClasses[class]; isRoot {
									warnings = append(warnings, Warning{
										Message: fmt.Sprintf("%s never matches: .%s is a root class, on <umono-root> only; use :scope.%s", text, class, class),
										Where:   text,
									})
									found = true
									break
								}
							}
							if found {
								break
							}
						}
					}
				}
				continue
			}
			walk(r.rules, inKeyframes || r.at == "keyframes")
		}
	}
	walk(rules, false)

	if len(warnings) == 0 {
		return nil
	}
	return warnings
}
