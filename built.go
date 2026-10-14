package themecore

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Built is built.toml, the manifest of a package. The CLI writes it from
// theme.toml during build: the keys of theme.toml are copied unchanged and
// theme-api-version and built-at are added.
type Built struct {
	Manifest
	ThemeAPIVersion int       `toml:"theme-api-version"`
	BuiltAt         time.Time `toml:"built-at"`
}

// ParseBuilt parses and validates built.toml.
func ParseBuilt(b []byte) (Built, error) {
	return parseBuilt(b, false)
}

// parseBuilt parses and validates built.toml. With dev, it is the built.toml
// of dev/dev (validateManifest).
func parseBuilt(b []byte, dev bool) (Built, error) {
	const file = "built.toml"
	doc, err := readSchema(file, b, schemaBuiltV1)
	if err != nil {
		return Built{}, err
	}
	if _, ok := doc["theme-api-version"]; !ok {
		return Built{}, fmt.Errorf("%s: theme-api-version: missing", file)
	}
	if _, ok := doc["theme-api-version"].(int64); !ok {
		return Built{}, fmt.Errorf("%s: theme-api-version: must be an integer", file)
	}
	if _, ok := doc["built-at"]; !ok {
		return Built{}, fmt.Errorf("%s: built-at: missing", file)
	}
	t, ok := doc["built-at"].(time.Time)
	if !ok || t.Location() != time.UTC {
		return Built{}, fmt.Errorf("%s: built-at: must be an offset date-time in UTC", file)
	}
	var built Built
	if err := decodeTOML(b, &built); err != nil {
		return Built{}, fmt.Errorf("%s: %w", file, err)
	}
	if err := validateManifest(file, doc, built.Manifest, dev); err != nil {
		return Built{}, err
	}
	if built.ThemeAPIVersion < 1 {
		return Built{}, fmt.Errorf("%s: theme-api-version: must be 1 or greater", file)
	}
	return built, nil
}

// CheckInstallDir checks that the install path dir, themes/{org}/{id}/{version}
// or themes-pending/{org}/{id}/{version}, matches id and version (built.toml,
// Karar 5). Only the last three elements of dir are compared.
func (b Built) CheckInstallDir(dir string) error {
	cleaned := filepath.ToSlash(filepath.Clean(dir))
	parts := strings.Split(cleaned, "/")
	nonEmpty := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) < 3 {
		return fmt.Errorf("built.toml: id and version do not match the install path %s", dir)
	}
	last3 := strings.Join(nonEmpty[len(nonEmpty)-3:], "/")
	want := b.ID + "/" + b.Version
	if last3 != want {
		return fmt.Errorf("built.toml: id and version do not match the install path %s", dir)
	}
	return nil
}

// checkContract checks that contract matches the id and version of the
// package's contract.toml (built.toml, Karar 6).
func (b Built) checkContract(id, version string) error {
	if b.Contract != id+"@"+version {
		return fmt.Errorf("built.toml: contract %s does not match contract.toml (%s@%s)", b.Contract, id, version)
	}
	return nil
}
