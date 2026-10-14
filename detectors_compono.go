package themecore

import (
	"fmt"
	"strings"

	"github.com/umono-cms/compono"
)

// contractFacts returns the analyses of the sources of the package's
// contract.toml and the names the contract defines as components: its
// [[components]] and the locals of every source. ok is false if the package
// has no contract.toml or it cannot be decoded. Detectors run on validated
// packages, so the file is only decoded.
func contractFacts(p Package) (sources []contractSource, defined map[string]bool, ok bool) {
	b, has := p.Files["contract.toml"]
	if !has {
		return nil, nil, false
	}
	var c Contract
	if err := decodeTOML(b, &c); err != nil {
		return nil, nil, false
	}
	sources = analyzeContract(c)
	defined = contractDefinedNames(c, sources)
	return sources, defined, true
}

// contractWhere is the place of rng in the source s: the file, the key of
// the source and the line in the source.
func contractWhere(s contractSource, rng compono.Range) string {
	return fmt.Sprintf("contract.toml:%s:%d", s.key, rng.Start.Line)
}

// detectBuiltin returns a detector for the built-in name: the first
// reference in contract.toml that resolves to it (a call, a value or a
// default), then the first address of attributes.toml whose type filter is
// it. A name the contract defines as a component never resolves to the
// built-in.
func detectBuiltin(name string) DetectorFunc {
	byAddress := detectAddress(func(address string, _ map[string]string) bool {
		parts := strings.Split(address, " ")
		return len(parts) > 1 && parts[len(parts)-1] == name
	})
	return func(p Package) (string, bool) {
		sources, defined, ok := contractFacts(p)
		if defined[name] {
			return "", false
		}
		if ok {
			for _, s := range sources {
				for _, ref := range s.analysis.Refs {
					if ref.Name == name && !ref.Local {
						return contractWhere(s, ref.Range), true
					}
				}
			}
		}
		return byAddress(p)
	}
}

// detectMarkdown returns a detector for the Markdown element name: its first
// use in contract.toml.
func detectMarkdown(name string) DetectorFunc {
	return func(p Package) (string, bool) {
		sources, _, ok := contractFacts(p)
		if !ok {
			return "", false
		}
		for _, s := range sources {
			for _, e := range s.analysis.Elements {
				if e.Name == name {
					return contractWhere(s, e.Range), true
				}
			}
		}
		return "", false
	}
}
