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

	if _, ok := doc["id"]; !ok {
		return Manifest{}, fmt.Errorf("%s: id: missing", file)
	}
	if err := checkText(file, "id", m.ID); err != nil {
		return Manifest{}, err
	}
	if !idPattern.MatchString(m.ID) {
		return Manifest{}, fmt.Errorf("%s: id: must be org/id in kebab-case", file)
	}
	if m.ID == "dev/dev" {
		return Manifest{}, fmt.Errorf("%s: id: dev/dev is reserved", file)
	}

	if _, ok := doc["contract"]; !ok {
		return Manifest{}, fmt.Errorf("%s: contract: missing", file)
	}
	if err := checkText(file, "contract", m.Contract); err != nil {
		return Manifest{}, err
	}
	if !contractRefPattern.MatchString(m.Contract) {
		return Manifest{}, fmt.Errorf("%s: contract: must be org/id@vMAJOR.MINOR", file)
	}

	if _, ok := doc["name"]; !ok {
		return Manifest{}, fmt.Errorf("%s: name: missing", file)
	}
	if err := checkText(file, "name", m.Name); err != nil {
		return Manifest{}, err
	}
	if m.Name == "" {
		return Manifest{}, fmt.Errorf("%s: name: must not be empty", file)
	}
	if err := checkMaxRunes(file, "name", m.Name, maxNameLen); err != nil {
		return Manifest{}, err
	}

	if _, ok := doc["version"]; !ok {
		return Manifest{}, fmt.Errorf("%s: version: missing", file)
	}
	if err := checkText(file, "version", m.Version); err != nil {
		return Manifest{}, err
	}
	if !versionPattern.MatchString(m.Version) {
		return Manifest{}, fmt.Errorf("%s: version: must be MAJOR.MINOR.PATCH", file)
	}

	if m.Description != nil {
		if err := checkText(file, "description", *m.Description); err != nil {
			return Manifest{}, err
		}
		if err := checkMaxRunes(file, "description", *m.Description, maxDescriptionLen); err != nil {
			return Manifest{}, err
		}
	}
	if m.License != nil {
		if err := checkText(file, "license", *m.License); err != nil {
			return Manifest{}, err
		}
		if !validSPDX(*m.License) {
			return Manifest{}, fmt.Errorf("%s: license: not a valid SPDX license expression", file)
		}
	}
	if m.Homepage != nil {
		if err := checkHTTPSURL(file, "homepage", *m.Homepage); err != nil {
			return Manifest{}, err
		}
	}
	if m.Repository != nil {
		if err := checkHTTPSURL(file, "repository", *m.Repository); err != nil {
			return Manifest{}, err
		}
	}

	if rawAuthors, ok := doc["authors"]; ok {
		if len(m.Authors) == 0 {
			return Manifest{}, fmt.Errorf("%s: authors: must have at least one author", file)
		}
		entries := rawAuthors.([]any)
		for i := range m.Authors {
			nameKey := fmt.Sprintf("authors[%d].name", i)
			urlKey := fmt.Sprintf("authors[%d].url", i)
			raw := entries[i].(map[string]any)
			if _, ok := raw["name"]; !ok {
				return Manifest{}, fmt.Errorf("%s: %s: missing", file, nameKey)
			}
			name := m.Authors[i].Name
			if err := checkText(file, nameKey, name); err != nil {
				return Manifest{}, err
			}
			if name == "" {
				return Manifest{}, fmt.Errorf("%s: %s: must not be empty", file, nameKey)
			}
			if err := checkMaxRunes(file, nameKey, name, maxNameLen); err != nil {
				return Manifest{}, err
			}
			if m.Authors[i].URL != nil {
				if err := checkHTTPSURL(file, urlKey, *m.Authors[i].URL); err != nil {
					return Manifest{}, err
				}
			}
		}
	}

	return m, nil
}
