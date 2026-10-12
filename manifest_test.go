package themecore

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func ptr(s string) *string {
	return &s
}

func manifestDoc(set map[string]string, extra string) string {
	base := []struct {
		key, value string
	}{
		{"schema", `"https://umono.dev/schemas/theme-manifest/v1"`},
		{"id", `"acme/minimal"`},
		{"contract", `"umono/e-commerce@v1.0"`},
		{"name", `"Minimal E-commerce"`},
		{"version", `"0.1.0"`},
	}
	baseKeys := make(map[string]struct{}, len(base))
	for _, b := range base {
		baseKeys[b.key] = struct{}{}
	}

	var sb strings.Builder
	for _, b := range base {
		if v, ok := set[b.key]; ok {
			if v == "-" {
				continue
			}
			sb.WriteString(b.key)
			sb.WriteString(" = ")
			sb.WriteString(v)
			sb.WriteByte('\n')
		} else {
			sb.WriteString(b.key)
			sb.WriteString(" = ")
			sb.WriteString(b.value)
			sb.WriteByte('\n')
		}
	}

	var extras []string
	for k := range set {
		if _, ok := baseKeys[k]; !ok {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		v := set[k]
		if v == "-" {
			continue
		}
		sb.WriteString(k)
		sb.WriteString(" = ")
		sb.WriteString(v)
		sb.WriteByte('\n')
	}

	sb.WriteString(extra)
	return sb.String()
}

func TestParseManifestExample(t *testing.T) {
	doc := `schema = "https://umono.dev/schemas/theme-manifest/v1"
id = "acme/minimal"
contract = "umono/e-commerce@v1.0"
name = "Minimal E-commerce"
version = "0.1.0"
description = "Clean e-commerce theme for Umono."
license = "MIT"
homepage = "https://example.com/minimal"
repository = "https://example.com/minimal.git"

[[authors]]
name = "Acme"
url = "https://example.com"
`
	m, err := ParseManifest([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Manifest{
		Schema:      "https://umono.dev/schemas/theme-manifest/v1",
		ID:          "acme/minimal",
		Contract:    "umono/e-commerce@v1.0",
		Name:        "Minimal E-commerce",
		Version:     "0.1.0",
		Description: ptr("Clean e-commerce theme for Umono."),
		License:     ptr("MIT"),
		Homepage:    ptr("https://example.com/minimal"),
		Repository:  ptr("https://example.com/minimal.git"),
		Authors:     []Author{{Name: "Acme", URL: ptr("https://example.com")}},
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("manifest mismatch:\n got: %#v\nwant: %#v", m, want)
	}
}

func TestParseManifestMinimal(t *testing.T) {
	doc := manifestDoc(nil, "")
	m, err := ParseManifest([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Description != nil {
		t.Errorf("Description: got %v, want nil", *m.Description)
	}
	if m.License != nil {
		t.Errorf("License: got %v, want nil", *m.License)
	}
	if m.Homepage != nil {
		t.Errorf("Homepage: got %v, want nil", *m.Homepage)
	}
	if m.Repository != nil {
		t.Errorf("Repository: got %v, want nil", *m.Repository)
	}
	if m.Authors != nil {
		t.Errorf("Authors: got %v, want nil", m.Authors)
	}
}

func TestParseManifestLicense(t *testing.T) {
	cases := []struct {
		expr    string
		wantErr string
	}{
		{"MIT", ""},
		{"MIT OR Apache-2.0", ""},
		{"LicenseRef-Acme", ""},
		{"(MIT AND BSD-3-Clause) OR Apache-2.0", ""},
		{"GPL-2.0-or-later WITH Classpath-exception-2.0", ""},
		{"GPL-2.0+", ""},
		{"DocumentRef-spdx-tool-1.2:LicenseRef-MIT-Style-2", ""},
		{"((MIT))", ""},
		{"", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT OR", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT or Apache-2.0", "theme.toml: license: not a valid SPDX license expression"},
		{"(MIT | MIT)", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT Apache-2.0", "theme.toml: license: not a valid SPDX license expression"},
		{"(MIT OR Apache-2.0) WITH Classpath-exception-2.0", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT WITH", "theme.toml: license: not a valid SPDX license expression"},
		{"AND", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT+ +", "theme.toml: license: not a valid SPDX license expression"},
		{"LicenseRef-", "theme.toml: license: not a valid SPDX license expression"},
		{"MIT  OR Apache-2.0", "theme.toml: license: not a valid SPDX license expression"},
		{" MIT", "theme.toml: license: has leading or trailing whitespace"},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			doc := manifestDoc(map[string]string{"license": `"` + c.expr + `"`}, "")
			_, err := ParseManifest([]byte(doc))
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("expected error containing %q, got %q", c.wantErr, err.Error())
			}
		})
	}
}

func TestParseManifestLengths(t *testing.T) {
	homepageBase := "https://example.com/" + strings.Repeat("a", 2048-20)
	if got := len(homepageBase); got != 2048 {
		t.Fatalf("homepage base length: got %d, want 2048", got)
	}

	cases := []struct {
		name    string
		key     string
		value   string
		wantErr string
	}{
		{"name at limit", "name", strings.Repeat("ş", 64), ""},
		{"name over limit", "name", strings.Repeat("ş", 65), "theme.toml: name: longer than 64 characters"},
		{"description at limit", "description", strings.Repeat("ş", 280), ""},
		{"description over limit", "description", strings.Repeat("ş", 281), "theme.toml: description: longer than 280 characters"},
		{"homepage at limit", "homepage", homepageBase, ""},
		{"homepage over limit", "homepage", homepageBase + "a", "theme.toml: homepage: longer than 2048 bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := manifestDoc(map[string]string{c.key: `"` + c.value + `"`}, "")
			_, err := ParseManifest([]byte(doc))
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("expected error containing %q, got %q", c.wantErr, err.Error())
			}
		})
	}
}

func TestParseManifestRejects(t *testing.T) {
	cases := []struct {
		name  string
		set   map[string]string
		extra string
		want  string
	}{
		{"schema missing", map[string]string{"schema": "-"}, "", "theme.toml: schema: missing"},
		{"schema not a string", map[string]string{"schema": "1"}, "", "theme.toml: schema: must be a string"},
		{"schema v2", map[string]string{"schema": `"https://umono.dev/schemas/theme-manifest/v2"`}, "", `theme.toml: schema: unsupported schema "https://umono.dev/schemas/theme-manifest/v2"`},
		{"schema of built.toml", map[string]string{"schema": `"https://umono.dev/schemas/theme-built/v1"`}, "", `unsupported schema "https://umono.dev/schemas/theme-built/v1"`},
		{"unknown key", map[string]string{"foo": `"x"`}, "", "theme.toml: unknown key foo"},
		{"built.toml key", map[string]string{"theme-api-version": "1"}, "", "theme.toml: unknown key theme-api-version"},
		{"unknown author key", nil, `
[[authors]]
name = "A"
email = "a@example.com"
`, "theme.toml: unknown key authors.email"},
		{"wrong type", map[string]string{"name": "5"}, "", "theme.toml:"},
		{"id missing", map[string]string{"id": "-"}, "", "theme.toml: id: missing"},
		{"id without org", map[string]string{"id": `"acme"`}, "", "theme.toml: id: must be org/id in kebab-case"},
		{"id upper case", map[string]string{"id": `"Acme/minimal"`}, "", "theme.toml: id: must be org/id in kebab-case"},
		{"id leading hyphen", map[string]string{"id": `"acme/-minimal"`}, "", "theme.toml: id: must be org/id in kebab-case"},
		{"id dev/dev", map[string]string{"id": `"dev/dev"`}, "", "theme.toml: id: dev/dev is reserved"},
		{"contract without v", map[string]string{"contract": `"umono/blog@1.0"`}, "", "theme.toml: contract: must be org/id@vMAJOR.MINOR"},
		{"contract leading zero", map[string]string{"contract": `"umono/blog@v01.0"`}, "", "theme.toml: contract: must be org/id@vMAJOR.MINOR"},
		{"contract without minor", map[string]string{"contract": `"umono/blog@v1"`}, "", "theme.toml: contract: must be org/id@vMAJOR.MINOR"},
		{"contract missing", map[string]string{"contract": "-"}, "", "theme.toml: contract: missing"},
		{"name empty", map[string]string{"name": `""`}, "", "theme.toml: name: must not be empty"},
		{"name missing", map[string]string{"name": "-"}, "", "theme.toml: name: missing"},
		{"name leading space", map[string]string{"name": `" Minimal"`}, "", "theme.toml: name: has leading or trailing whitespace"},
		{"name trailing space", map[string]string{"name": `"Minimal "`}, "", "theme.toml: name: has leading or trailing whitespace"},
		{"name control character", map[string]string{"name": `"Mini\u0007mal"`}, "", "theme.toml: name: contains a control character"},
		{"name newline", map[string]string{"name": `"Mini\nmal"`}, "", "theme.toml: name: contains a control character"},
		{"version with v", map[string]string{"version": `"v0.1.0"`}, "", "theme.toml: version: must be MAJOR.MINOR.PATCH"},
		{"version without patch", map[string]string{"version": `"0.1"`}, "", "theme.toml: version: must be MAJOR.MINOR.PATCH"},
		{"version leading zero", map[string]string{"version": `"01.0.0"`}, "", "theme.toml: version: must be MAJOR.MINOR.PATCH"},
		{"version pre-release", map[string]string{"version": `"0.1.0-beta"`}, "", "theme.toml: version: must be MAJOR.MINOR.PATCH"},
		{"version build metadata", map[string]string{"version": `"0.1.0+x"`}, "", "theme.toml: version: must be MAJOR.MINOR.PATCH"},
		{"version missing", map[string]string{"version": "-"}, "", "theme.toml: version: missing"},
		{"license open paren", map[string]string{"license": `"(MIT"`}, "", "theme.toml: license: not a valid SPDX license expression"},
		{"license close paren", map[string]string{"license": `"MIT)"`}, "", "theme.toml: license: not a valid SPDX license expression"},
		{"homepage http", map[string]string{"homepage": `"http://example.com"`}, "", "theme.toml: homepage: must be an absolute https URL"},
		{"homepage no host", map[string]string{"homepage": `"https://"`}, "", "theme.toml: homepage: must be an absolute https URL"},
		{"homepage relative", map[string]string{"homepage": `"example.com"`}, "", "theme.toml: homepage: must be an absolute https URL"},
		{"homepage upper case scheme", map[string]string{"homepage": `"HTTPS://example.com"`}, "", "theme.toml: homepage: must be an absolute https URL"},
		{"repository http", map[string]string{"repository": `"http://example.com/a.git"`}, "", "theme.toml: repository: must be an absolute https URL"},
		{"authors empty", map[string]string{"authors": "[]"}, "", "theme.toml: authors: must have at least one author"},
		{"author name missing", nil, `
[[authors]]
url = "https://example.com"
`, "theme.toml: authors[0].name: missing"},
		{"author name empty", nil, `
[[authors]]
name = ""
`, "theme.toml: authors[0].name: must not be empty"},
		{"second author url http", nil, `
[[authors]]
name = "A"

[[authors]]
name = "B"
url = "http://example.com"
`, "theme.toml: authors[1].url: must be an absolute https URL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(manifestDoc(c.set, c.extra)))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %q", c.want, err.Error())
			}
		})
	}
}
