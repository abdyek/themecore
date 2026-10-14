package themecore

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func contractDoc(head map[string]string, tables string) string {
	base := []struct {
		key, value string
	}{
		{"schema", `"https://umono.dev/schemas/theme-contract/v1"`},
		{"id", `"example/blog"`},
		{"version", `"v0.1"`},
	}
	baseKeys := make(map[string]struct{}, len(base))
	for _, b := range base {
		baseKeys[b.key] = struct{}{}
	}

	var sb strings.Builder
	for _, b := range base {
		if v, ok := head[b.key]; ok {
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
	for k := range head {
		if _, ok := baseKeys[k]; !ok {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		v := head[k]
		if v == "-" {
			continue
		}
		sb.WriteString(k)
		sb.WriteString(" = ")
		sb.WriteString(v)
		sb.WriteByte('\n')
	}

	if tables == "" {
		tables = `[[layouts]]
id = "post"
source = "# {{ title }}"`
	}
	sb.WriteByte('\n')
	sb.WriteString(tables)
	return sb.String()
}

func layoutTable(lines ...string) string {
	return "[[layouts]]\n" + strings.Join(lines, "\n")
}

func componentTable(lines ...string) string {
	return "[[components]]\n" + strings.Join(lines, "\n")
}

func TestParseContractExample(t *testing.T) {
	doc := `schema = "https://umono.dev/schemas/theme-contract/v1"
id = "example/blog"
version = "v0.1"

[[layouts]]
id = "post"
description = "A single blog post"
source = '''
~ title = "" body = NO_MATTER
# {{ title }}
{{ POST_BODY body = body }}
'''

[[components]]
name = "POST_BODY"
expose = true
source = '''
~ body = NO_MATTER
{{ body }}
'''
`
	c, err := ParseContract([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Contract{
		Schema:      "https://umono.dev/schemas/theme-contract/v1",
		ID:          "example/blog",
		Version:     "v0.1",
		Description: nil,
		Layouts: []Layout{
			{
				ID:          "post",
				Description: ptr("A single blog post"),
				Source:      "~ title = \"\" body = NO_MATTER\n# {{ title }}\n{{ POST_BODY body = body }}\n",
			},
		},
		Components: []Component{
			{
				Name:   "POST_BODY",
				Source: "~ body = NO_MATTER\n{{ body }}\n",
				Expose: true,
			},
		},
	}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("contract mismatch:\n got: %#v\nwant: %#v", c, want)
	}
}

func TestParseContractMinimal(t *testing.T) {
	doc := contractDoc(nil, "")
	c, err := ParseContract([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Components != nil {
		t.Errorf("Components: got %v, want nil", c.Components)
	}
	if len(c.Layouts) != 1 {
		t.Fatalf("Layouts: got %d layouts, want 1", len(c.Layouts))
	}
	if c.Layouts[0].ID != "post" {
		t.Errorf("Layouts[0].ID: got %q, want %q", c.Layouts[0].ID, "post")
	}
}

func TestParseContractRejects(t *testing.T) {
	cases := []struct {
		name   string
		head   map[string]string
		tables string
		want   string
	}{
		{"theme schema", map[string]string{"schema": `"https://umono.dev/schemas/theme-manifest/v1"`}, "", `contract.toml: schema: unsupported schema "https://umono.dev/schemas/theme-manifest/v1"`},
		{"id missing", map[string]string{"id": "-"}, "", "contract.toml: id: missing"},
		{"id", map[string]string{"id": `"blog"`}, "", "contract.toml: id: must be org/id in kebab-case"},
		{"version missing", map[string]string{"version": "-"}, "", "contract.toml: version: missing"},
		{"version without v", map[string]string{"version": `"0.1"`}, "", "contract.toml: version: must be vMAJOR.MINOR"},
		{"version with patch", map[string]string{"version": `"v0.1.0"`}, "", "contract.toml: version: must be vMAJOR.MINOR"},
		{"version leading zero", map[string]string{"version": `"v0.01"`}, "", "contract.toml: version: must be vMAJOR.MINOR"},
		{"description", map[string]string{"description": `"` + strings.Repeat("a", 281) + `"`}, "", "contract.toml: description: longer than 280 characters"},
		{"unknown key", map[string]string{"foo": `"x"`}, "", "contract.toml: unknown key foo"},
		{"layouts missing", nil, componentTable(`name = "A"`, `source = "x"`), "contract.toml: layouts: missing"},
		{"layouts empty", map[string]string{"layouts": "[]"}, componentTable(`name = "A"`, `source = "x"`), "contract.toml: layouts: must have at least one layout"},
		{"layout id missing", nil, layoutTable(`source = "x"`), "contract.toml: layouts[0].id: missing"},
		{"layout id", nil, layoutTable(`id = "Post"`, `source = "x"`), "contract.toml: layouts[0].id: must be kebab-case"},
		{"layout id duplicate", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + layoutTable(`id = "post"`, `source = "y"`), "contract.toml: layouts[1].id: duplicate id post"},
		{"layout description", nil, layoutTable(`id = "post"`, `description = " x"`, `source = "x"`), "contract.toml: layouts[0].description: has leading or trailing whitespace"},
		{"layout source missing", nil, layoutTable(`id = "post"`), "contract.toml: layouts[0].source: missing"},
		{"layout source empty", nil, layoutTable(`id = "post"`, `source = ""`), "contract.toml: layouts[0].source: must not be empty"},
		{"layout source blank", nil, layoutTable(`id = "post"`, `source = " \n\t "`), "contract.toml: layouts[0].source: must not be empty"},
		{"layout source control", nil, layoutTable(`id = "post"`, `source = "a\u0000b"`), "contract.toml: layouts[0].source: contains a control character"},
		{"layout unknown key", nil, layoutTable(`id = "post"`, `source = "x"`, `expose = true`), "contract.toml: unknown key layouts.expose"},
		{"component name missing", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + componentTable(`source = "x"`), "contract.toml: components[0].name: missing"},
		{"component name", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + componentTable(`name = "PostBody"`, `source = "x"`), "contract.toml: components[0].name: must be SCREAMING_SNAKE_CASE"},
		{"component name duplicate", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + componentTable(`name = "A"`, `source = "x"`) + "\n\n" + componentTable(`name = "A"`, `source = "y"`), "contract.toml: components[1].name: duplicate name A"},
		{"component source missing", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + componentTable(`name = "A"`), "contract.toml: components[0].source: missing"},
		{"component expose type", nil, layoutTable(`id = "post"`, `source = "x"`) + "\n\n" + componentTable(`name = "A"`, `source = "x"`, `expose = "yes"`), "contract.toml:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseContract([]byte(contractDoc(tc.head, tc.tables)))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestParseContractSourceWhitespace(t *testing.T) {
	doc := contractDoc(nil, layoutTable(`id = "post"`, `source = "a\r\n\tb\n"`))
	c, err := ParseContract([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Layouts) != 1 {
		t.Fatalf("Layouts: got %d layouts, want 1", len(c.Layouts))
	}
	want := "a\r\n\tb\n"
	if c.Layouts[0].Source != want {
		t.Errorf("Layouts[0].Source: got %q, want %q", c.Layouts[0].Source, want)
	}
}

func TestParseContractRejectsSourceSyntaxErrors(t *testing.T) {
	post := "id = " + string(rune(34)) + "post" + string(rune(34))
	cases := []struct {
		name   string
		tables string
		want   string
	}{
		{
			name:   "invalid expression in layout",
			tables: layoutTable(post, `source = '{{ IF x }}'`),
			want:   "contract.toml: layouts[0].source: 1:1: invalid-expression",
		},
		{
			name:   "unclosed expression in layout",
			tables: layoutTable(post, `source = 'a {{ b'`),
			want:   "contract.toml: layouts[0].source: 1:3: unclosed-expression",
		},
		{
			name:   "invalid expression in uncalled local",
			tables: layoutTable(post, "source = '''\nok\n\n~ X\n{{ Bad }}\n'''"),
			want:   "contract.toml: layouts[0].source: 4:1: invalid-expression",
		},
		{
			name:   "invalid parameter definition in component",
			tables: layoutTable(post, `source = 'x'`) + "\n\n" + componentTable(`name = "A"`, "source = '''\n~ y\n{{ y }}\n'''"),
			want:   "contract.toml: components[0].source: 1:3: invalid-parameter-definition",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseContract([]byte(contractDoc(nil, tc.tables)))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
			if !errors.Is(err, ErrOutdated) {
				t.Errorf("error %q does not wrap ErrOutdated", err.Error())
			}
		})
	}
}

func TestParseContractAcceptsValueDependentErrors(t *testing.T) {
	tables := layoutTable("id = "+string(rune(34))+"post"+string(rune(34)), "source = '''\n{{ title }}\n{{ GLOBAL_OR_COMPONENT }}\n'''")
	if _, err := ParseContract([]byte(contractDoc(nil, tables))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseContractRejectsUndefinedCalls(t *testing.T) {
	post := "id = " + string(rune(34)) + "post" + string(rune(34))
	compA := "name = " + string(rune(34)) + "A" + string(rune(34))
	cases := []struct {
		name   string
		tables string
		want   string
	}{
		{
			name:   "undefined call in layout",
			tables: layoutTable(post, `source = '{{ NOPE }}'`),
			want:   "contract.toml: layouts[0].source: 1:1: undefined component NOPE",
		},
		{
			name:   "undefined value in layout",
			tables: layoutTable(post, `source = '{{ A c = NOPE }}'`) + "\n\n" + componentTable(compA, "source = '''\n~ c = NO_MATTER\n{{ c }}\n'''"),
			want:   "contract.toml: layouts[0].source: 1:10: undefined component NOPE",
		},
		{
			name:   "local of another source",
			tables: layoutTable(post, `source = '{{ X }}'`) + "\n\n" + componentTable(compA, "source = '''\nx\n\n~ X\ny\n'''"),
			want:   "contract.toml: layouts[0].source: 1:1: undefined component X",
		},
		{
			name:   "undefined call in component",
			tables: layoutTable(post, `source = '{{ A }}'`) + "\n\n" + componentTable(compA, `source = '{{ NOPE }}'`),
			want:   "contract.toml: components[0].source: 1:1: undefined component NOPE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseContract([]byte(contractDoc(nil, tc.tables)))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
			if !errors.Is(err, ErrOutdated) {
				t.Errorf("error %q does not wrap ErrOutdated", err.Error())
			}
		})
	}
}

func TestParseContractAcceptsDefinedCalls(t *testing.T) {
	post := "id = " + string(rune(34)) + "post" + string(rune(34))
	compA := "name = " + string(rune(34)) + "A" + string(rune(34))
	tables := layoutTable(post, "source = '''\n~ c = NOPE\n{{ c }}\n{{ MAIN content = A }}\n{{ L }}\n\n~ L\nl\n'''") +
		"\n\n" + componentTable(compA, `source = '{{ A }}'`)
	if _, err := ParseContract([]byte(contractDoc(nil, tables))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
