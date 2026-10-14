package themecore

import (
	"fmt"
)

// Author is an entry of the authors array.
type Author struct {
	Name string  `toml:"name"`
	URL  *string `toml:"url"`
}

// Manifest is theme.toml, the manifest the theme author writes. The CLI reads
// it; it does not go into the package. Optional keys are nil when absent.
type Manifest struct {
	Schema      string   `toml:"schema"`
	ID          string   `toml:"id"`
	Contract    string   `toml:"contract"`
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description *string  `toml:"description"`
	License     *string  `toml:"license"`
	Homepage    *string  `toml:"homepage"`
	Repository  *string  `toml:"repository"`
	Authors     []Author `toml:"authors"`
}

// ParseManifest parses and validates theme.toml.
func ParseManifest(b []byte) (Manifest, error) {
	const file = "theme.toml"
	doc, err := readSchema(file, b, schemaManifestV1)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := decodeTOML(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", file, err)
	}
	if err := validateManifest(file, doc, m, false); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// validateManifest checks the keys that theme.toml and built.toml share. doc
// is the decoded document, which tells which keys are present; file is the
// file name in messages. With dev, the manifest is that of dev/dev, the theme
// of Umono's dev mode: the id must be dev/dev and the version 0.0.0.
func validateManifest(file string, doc map[string]any, m Manifest, dev bool) error {
	if _, ok := doc["id"]; !ok {
		return fmt.Errorf("%s: id: missing", file)
	}
	if err := checkText(file, "id", m.ID); err != nil {
		return err
	}
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("%s: id: must be org/id in kebab-case", file)
	}
	if dev && m.ID != "dev/dev" {
		return fmt.Errorf("%s: id: must be dev/dev in dev mode", file)
	}
	if !dev && m.ID == "dev/dev" {
		return fmt.Errorf("%s: id: dev/dev is reserved", file)
	}

	if _, ok := doc["contract"]; !ok {
		return fmt.Errorf("%s: contract: missing", file)
	}
	if err := checkText(file, "contract", m.Contract); err != nil {
		return err
	}
	if !contractRefPattern.MatchString(m.Contract) {
		return fmt.Errorf("%s: contract: must be org/id@vMAJOR.MINOR", file)
	}

	if _, ok := doc["name"]; !ok {
		return fmt.Errorf("%s: name: missing", file)
	}
	if err := checkText(file, "name", m.Name); err != nil {
		return err
	}
	if m.Name == "" {
		return fmt.Errorf("%s: name: must not be empty", file)
	}
	if err := checkMaxRunes(file, "name", m.Name, maxNameLen); err != nil {
		return err
	}

	if _, ok := doc["version"]; !ok {
		return fmt.Errorf("%s: version: missing", file)
	}
	if err := checkText(file, "version", m.Version); err != nil {
		return err
	}
	if !versionPattern.MatchString(m.Version) {
		return fmt.Errorf("%s: version: must be MAJOR.MINOR.PATCH", file)
	}
	if dev && m.Version != "0.0.0" {
		return fmt.Errorf("%s: version: must be 0.0.0 in dev mode", file)
	}

	if m.Description != nil {
		if err := checkText(file, "description", *m.Description); err != nil {
			return err
		}
		if err := checkMaxRunes(file, "description", *m.Description, maxDescriptionLen); err != nil {
			return err
		}
	}
	if m.License != nil {
		if err := checkText(file, "license", *m.License); err != nil {
			return err
		}
		if !validSPDX(*m.License) {
			return fmt.Errorf("%s: license: not a valid SPDX license expression", file)
		}
	}
	if m.Homepage != nil {
		if err := checkHTTPSURL(file, "homepage", *m.Homepage); err != nil {
			return err
		}
	}
	if m.Repository != nil {
		if err := checkHTTPSURL(file, "repository", *m.Repository); err != nil {
			return err
		}
	}

	if rawAuthors, ok := doc["authors"]; ok {
		if len(m.Authors) == 0 {
			return fmt.Errorf("%s: authors: must have at least one author", file)
		}
		entries := rawAuthors.([]any)
		for i := range m.Authors {
			nameKey := fmt.Sprintf("authors[%d].name", i)
			urlKey := fmt.Sprintf("authors[%d].url", i)
			raw := entries[i].(map[string]any)
			if _, ok := raw["name"]; !ok {
				return fmt.Errorf("%s: %s: missing", file, nameKey)
			}
			name := m.Authors[i].Name
			if err := checkText(file, nameKey, name); err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("%s: %s: must not be empty", file, nameKey)
			}
			if err := checkMaxRunes(file, nameKey, name, maxNameLen); err != nil {
				return err
			}
			if m.Authors[i].URL != nil {
				if err := checkHTTPSURL(file, urlKey, *m.Authors[i].URL); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
