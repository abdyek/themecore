package themecore

import (
	"fmt"
)

// schemaManifestV1 is the schema URL of theme.toml.
const schemaManifestV1 = "https://umono.dev/schemas/theme-manifest/v1"

// schemaBuiltV1 is the schema URL of built.toml.
const schemaBuiltV1 = "https://umono.dev/schemas/theme-built/v1"

// schemaContractV1 is the schema URL of contract.toml.
const schemaContractV1 = "https://umono.dev/schemas/theme-contract/v1"

// readSchema is the schema check: it decodes b into a map and checks that its
// schema key is one of supported. The rest of the file is checked by the
// file's own validator; the map tells it which keys are present. Errors are
// prefixed with file.
func readSchema(file string, b []byte, supported ...string) (map[string]any, error) {
	var doc map[string]any
	if err := decodeTOML(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	v, ok := doc["schema"]
	if !ok {
		return nil, fmt.Errorf("%s: schema: missing", file)
	}
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("%s: schema: must be a string", file)
	}
	for _, c := range supported {
		if s == c {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("%s: schema: unsupported schema %q", file, s)
}
