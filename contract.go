package themecore

import (
	"fmt"
)

// Contract is contract.toml. Contracts are published on umono.dev and copied
// into every theme package unchanged.
type Contract struct {
	Schema      string      `toml:"schema"`
	ID          string      `toml:"id"`
	Version     string      `toml:"version"`
	Description *string     `toml:"description"`
	Layouts     []Layout    `toml:"layouts"`
	Components  []Component `toml:"components"`
}

// Layout is an entry of the layouts array.
type Layout struct {
	ID          string  `toml:"id"`
	Description *string `toml:"description"`
	Source      string  `toml:"source"`
}

// Component is an entry of the components array. Expose tells whether the
// component is a scope for addressing.
type Component struct {
	Name   string `toml:"name"`
	Source string `toml:"source"`
	Expose bool   `toml:"expose"`
}

// ParseContract parses and validates contract.toml: the structural layer
// (types, required keys, grammars and uniqueness), then the sources, which
// are parsed with Compono. A source with a grammar error, or a call that
// resolves to no local, contract component or built-in, is rejected with an
// error that wraps ErrOutdated.
func ParseContract(b []byte) (Contract, error) {
	const file = "contract.toml"
	doc, err := readSchema(file, b, schemaContractV1)
	if err != nil {
		return Contract{}, err
	}
	var c Contract
	if err := decodeTOML(b, &c); err != nil {
		return Contract{}, fmt.Errorf("%s: %w", file, err)
	}
	if _, ok := doc["id"]; !ok {
		return Contract{}, fmt.Errorf("%s: id: missing", file)
	}
	if err := checkText(file, "id", c.ID); err != nil {
		return Contract{}, err
	}
	if !idPattern.MatchString(c.ID) {
		return Contract{}, fmt.Errorf("%s: id: must be org/id in kebab-case", file)
	}
	if _, ok := doc["version"]; !ok {
		return Contract{}, fmt.Errorf("%s: version: missing", file)
	}
	if err := checkText(file, "version", c.Version); err != nil {
		return Contract{}, err
	}
	if !contractVersionPattern.MatchString(c.Version) {
		return Contract{}, fmt.Errorf("%s: version: must be vMAJOR.MINOR", file)
	}
	if c.Description != nil {
		if err := checkText(file, "description", *c.Description); err != nil {
			return Contract{}, err
		}
		if err := checkMaxRunes(file, "description", *c.Description, maxDescriptionLen); err != nil {
			return Contract{}, err
		}
	}
	if _, ok := doc["layouts"]; !ok {
		return Contract{}, fmt.Errorf("%s: layouts: missing", file)
	}
	if len(c.Layouts) == 0 {
		return Contract{}, fmt.Errorf("%s: layouts: must have at least one layout", file)
	}
	rawLayouts := doc["layouts"].([]any)
	seenLayouts := make(map[string]struct{}, len(c.Layouts))
	for i := range c.Layouts {
		idKey := fmt.Sprintf("layouts[%d].id", i)
		descKey := fmt.Sprintf("layouts[%d].description", i)
		sourceKey := fmt.Sprintf("layouts[%d].source", i)
		raw := rawLayouts[i].(map[string]any)
		if _, ok := raw["id"]; !ok {
			return Contract{}, fmt.Errorf("%s: %s: missing", file, idKey)
		}
		if err := checkText(file, idKey, c.Layouts[i].ID); err != nil {
			return Contract{}, err
		}
		if !kebabPattern.MatchString(c.Layouts[i].ID) {
			return Contract{}, fmt.Errorf("%s: %s: must be kebab-case", file, idKey)
		}
		if _, dup := seenLayouts[c.Layouts[i].ID]; dup {
			return Contract{}, fmt.Errorf("%s: %s: duplicate id %s", file, idKey, c.Layouts[i].ID)
		}
		seenLayouts[c.Layouts[i].ID] = struct{}{}
		if c.Layouts[i].Description != nil {
			if err := checkText(file, descKey, *c.Layouts[i].Description); err != nil {
				return Contract{}, err
			}
			if err := checkMaxRunes(file, descKey, *c.Layouts[i].Description, maxDescriptionLen); err != nil {
				return Contract{}, err
			}
		}
		if _, ok := raw["source"]; !ok {
			return Contract{}, fmt.Errorf("%s: %s: missing", file, sourceKey)
		}
		if err := checkSource(file, sourceKey, c.Layouts[i].Source); err != nil {
			return Contract{}, err
		}
	}
	if rawComponents, ok := doc["components"]; ok {
		rawComponentsList := rawComponents.([]any)
		seenComponents := make(map[string]struct{}, len(c.Components))
		for i := range c.Components {
			nameKey := fmt.Sprintf("components[%d].name", i)
			sourceKey := fmt.Sprintf("components[%d].source", i)
			raw := rawComponentsList[i].(map[string]any)
			if _, ok := raw["name"]; !ok {
				return Contract{}, fmt.Errorf("%s: %s: missing", file, nameKey)
			}
			if err := checkText(file, nameKey, c.Components[i].Name); err != nil {
				return Contract{}, err
			}
			if !componentNamePattern.MatchString(c.Components[i].Name) {
				return Contract{}, fmt.Errorf("%s: %s: must be SCREAMING_SNAKE_CASE", file, nameKey)
			}
			if _, dup := seenComponents[c.Components[i].Name]; dup {
				return Contract{}, fmt.Errorf("%s: %s: duplicate name %s", file, nameKey, c.Components[i].Name)
			}
			seenComponents[c.Components[i].Name] = struct{}{}
			if _, ok := raw["source"]; !ok {
				return Contract{}, fmt.Errorf("%s: %s: missing", file, sourceKey)
			}
			if err := checkSource(file, sourceKey, c.Components[i].Source); err != nil {
				return Contract{}, err
			}
		}
	}
	sources := analyzeContract(c)
	if err := checkContractSyntax(sources); err != nil {
		return Contract{}, err
	}
	if err := checkContractCalls(c, sources); err != nil {
		return Contract{}, err
	}
	return c, nil
}
