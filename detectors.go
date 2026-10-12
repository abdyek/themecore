package themecore

import (
	"fmt"
	"sort"
	"strings"
)

// decodeAddresses decodes the address table of attributes.toml for the
// detectors. It does not validate the file.
func decodeAddresses(b []byte) (map[string]map[string]string, error) {
	var v struct {
		Schema  string                       `toml:"schema"`
		Address map[string]map[string]string `toml:"address"`
	}
	if err := decodeTOML(b, &v); err != nil {
		return nil, err
	}
	return v.Address, nil
}

// attributeAddresses returns the address map of attributes.toml of p, or nil
// if the file is absent or cannot be decoded. Detectors run on validated
// packages; a file that does not decode shows no feature.
func attributeAddresses(p Package) map[string]map[string]string {
	b, ok := p.Files["attributes.toml"]
	if !ok {
		return nil
	}
	addresses, err := decodeAddresses(b)
	if err != nil {
		return nil
	}
	return addresses
}

// detectAddress returns a detector that scans the addresses of p in sorted
// order and returns the first address for which pred is true.
func detectAddress(pred func(address string, attrs map[string]string) bool) DetectorFunc {
	return func(p Package) (string, bool) {
		addrs := attributeAddresses(p)
		if len(addrs) == 0 {
			return "", false
		}
		keys := make([]string, 0, len(addrs))
		for k := range addrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, address := range keys {
			if pred(address, addrs[address]) {
				return address, true
			}
		}
		return "", false
	}
}

// addressParts returns the number of space-separated parts of address.
func addressParts(address string) int {
	return strings.Count(address, " ") + 1
}

// detectFile returns a detector that reports path if p.Files has path.
func detectFile(path string) DetectorFunc {
	return func(p Package) (string, bool) {
		if _, ok := p.Files[path]; ok {
			return path, true
		}
		return "", false
	}
}

// detectAsset returns a detector that scans the paths of p.Files in sorted
// order and returns the first one that starts with "assets/" and ends with ext.
func detectAsset(ext string) DetectorFunc {
	return func(p Package) (string, bool) {
		paths := make([]string, 0, len(p.Files))
		for path := range p.Files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			if strings.HasPrefix(path, "assets/") && strings.HasSuffix(path, ext) {
				return path, true
			}
		}
		return "", false
	}
}

// hasClass reports whether attrs has the "class" key.
func hasClass(_ string, attrs map[string]string) bool {
	_, ok := attrs["class"]
	return ok
}

// hasSizes reports whether attrs has the "sizes" key.
func hasSizes(_ string, attrs map[string]string) bool {
	_, ok := attrs["sizes"]
	return ok
}

// layoutAddress reports whether address has one part.
func layoutAddress(address string, _ map[string]string) bool {
	return addressParts(address) == 1
}

// layoutBuiltinAddress reports whether address has two parts.
func layoutBuiltinAddress(address string, _ map[string]string) bool {
	return addressParts(address) == 2
}

// layoutComponentBuiltinAddress reports whether address has three parts.
func layoutComponentBuiltinAddress(address string, _ map[string]string) bool {
	return addressParts(address) == 3
}

// findAtRuleLine walks rules and their children depth-first in source order
// (an at-rule before its children) and returns the line of the first rule
// whose at equals name.
func findAtRuleLine(rules []cssRule, name string) (int, bool) {
	for _, r := range rules {
		if r.at == name {
			return r.line, true
		}
		if len(r.rules) > 0 {
			if line, found := findAtRuleLine(r.rules, name); found {
				return line, true
			}
		}
	}
	return 0, false
}

// detectAtRule returns a detector for an at-rule in the package's canonical
// assets/theme.css. Detectors look at the canonical CSS, not the author's
// source. where is the file and the line of the first such at-rule.
func detectAtRule(name string) DetectorFunc {
	return func(p Package) (string, bool) {
		b, ok := p.Files[themeCSS]
		if !ok {
			return "", false
		}
		rules, err := parseCSS(b)
		if err != nil {
			return "", false
		}
		if line, found := findAtRuleLine(rules, name); found {
			return fmt.Sprintf("%s:%d", themeCSS, line), true
		}
		return "", false
	}
}
