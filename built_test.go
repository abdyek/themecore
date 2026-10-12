package themecore

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func builtDoc(set map[string]string, extra string) string {
	base := []struct {
		key, value string
	}{
		{"schema", `"https://umono.dev/schemas/theme-built/v1"`},
		{"id", `"acme/minimal"`},
		{"contract", `"umono/e-commerce@v1.0"`},
		{"theme-api-version", "1"},
		{"name", `"Minimal E-commerce"`},
		{"version", `"0.1.0"`},
		{"built-at", "2026-06-02T00:00:00Z"},
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

func TestParseBuiltExample(t *testing.T) {
	extra := `[[authors]]
name = "Acme"
url = "https://example.com"
`
	doc := builtDoc(map[string]string{
		"description": `"Clean e-commerce theme for Umono."`,
		"license":     `"MIT"`,
	}, extra)
	b, err := ParseBuilt([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Built{
		Manifest: Manifest{
			Schema:      "https://umono.dev/schemas/theme-built/v1",
			ID:          "acme/minimal",
			Contract:    "umono/e-commerce@v1.0",
			Name:        "Minimal E-commerce",
			Version:     "0.1.0",
			Description: ptr("Clean e-commerce theme for Umono."),
			License:     ptr("MIT"),
			Authors:     []Author{{Name: "Acme", URL: ptr("https://example.com")}},
		},
		ThemeAPIVersion: 1,
		BuiltAt:         time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("built mismatch:\n got: %#v\nwant: %#v", b, want)
	}
}

func TestParseBuiltZeroOffset(t *testing.T) {
	doc := builtDoc(map[string]string{
		"built-at": "2026-06-02T00:00:00+00:00",
	}, "")
	b, err := ParseBuilt([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b.BuiltAt.Equal(time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("BuiltAt: got %v, want %v", b.BuiltAt, time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC))
	}
}

func TestParseBuiltRejects(t *testing.T) {
	cases := []struct {
		name string
		set  map[string]string
		want string
	}{
		{"manifest schema", map[string]string{"schema": `"https://umono.dev/schemas/theme-manifest/v1"`}, `built.toml: schema: unsupported schema "https://umono.dev/schemas/theme-manifest/v1"`},
		{"api version missing", map[string]string{"theme-api-version": "-"}, "built.toml: theme-api-version: missing"},
		{"api version zero", map[string]string{"theme-api-version": "0"}, "built.toml: theme-api-version: must be 1 or greater"},
		{"api version float", map[string]string{"theme-api-version": "1.0"}, "built.toml: theme-api-version: must be an integer"},
		{"api version string", map[string]string{"theme-api-version": `"1"`}, "built.toml: theme-api-version: must be an integer"},
		{"built-at missing", map[string]string{"built-at": "-"}, "built.toml: built-at: missing"},
		{"built-at local", map[string]string{"built-at": "2026-06-02T00:00:00"}, "built.toml: built-at: must be an offset date-time in UTC"},
		{"built-at offset", map[string]string{"built-at": "2026-06-02T03:00:00+03:00"}, "built.toml: built-at: must be an offset date-time in UTC"},
		{"built-at date", map[string]string{"built-at": "2026-06-02"}, "built.toml: built-at: must be an offset date-time in UTC"},
		{"built-at string", map[string]string{"built-at": `"2026-06-02T00:00:00Z"`}, "built.toml: built-at: must be an offset date-time in UTC"},
		{"dev/dev", map[string]string{"id": `"dev/dev"`}, "built.toml: id: dev/dev is reserved"},
		{"name empty", map[string]string{"name": `""`}, "built.toml: name: must not be empty"},
		{"unknown key", map[string]string{"foo": `"x"`}, "built.toml: unknown key foo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBuilt([]byte(builtDoc(c.set, "")))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %q", c.want, err.Error())
			}
		})
	}
}

func TestBuiltCheckInstallDir(t *testing.T) {
	b, err := ParseBuilt([]byte(builtDoc(nil, "")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ok := []string{
		"themes/acme/minimal/0.1.0",
		"/srv/site/themes-pending/acme/minimal/0.1.0/",
		filepath.Join("themes", "acme", "minimal", "0.1.0"),
	}
	for _, dir := range ok {
		if err := b.CheckInstallDir(dir); err != nil {
			t.Errorf("CheckInstallDir(%q): unexpected error: %v", dir, err)
		}
	}
	bad := []string{
		"themes/acme/minimal/0.2.0",
		"themes/acme/other/0.1.0",
		"themes/other/minimal/0.1.0",
		"minimal/0.1.0",
	}
	for _, dir := range bad {
		err := b.CheckInstallDir(dir)
		if err == nil {
			t.Errorf("CheckInstallDir(%q): expected error, got nil", dir)
			continue
		}
		if !strings.Contains(err.Error(), "built.toml: id and version do not match the install path") {
			t.Errorf("CheckInstallDir(%q): expected error containing %q, got %q", dir, "built.toml: id and version do not match the install path", err.Error())
		}
	}
}

func TestBuiltCheckContract(t *testing.T) {
	b, err := ParseBuilt([]byte(builtDoc(nil, "")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.checkContract("umono/e-commerce", "v1.0"); err != nil {
		t.Errorf("checkContract matching: unexpected error: %v", err)
	}
	err = b.checkContract("umono/e-commerce", "v1.1")
	if err == nil {
		t.Fatalf("checkContract version mismatch: expected error, got nil")
	}
	if got, want := err.Error(), "built.toml: contract umono/e-commerce@v1.0 does not match contract.toml (umono/e-commerce@v1.1)"; got != want {
		t.Errorf("checkContract version mismatch: got %q, want %q", got, want)
	}
	if err := b.checkContract("umono/blog", "v1.0"); err == nil {
		t.Errorf("checkContract id mismatch: expected non-nil error, got nil")
	}
}
