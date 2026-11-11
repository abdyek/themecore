package themecore

import (
	"reflect"
	"strings"
	"testing"
)

func TestDecodeTOMLRejects(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		v    any
		want string
	}{
		{
			name: "TOML v1.1.0 syntax",
			doc:  "a = { b = 1, }\n",
			v:    new(map[string]any),
			want: "",
		},
		{
			name: "unknown key",
			doc:  "a = 1\nb = 2\n",
			v: &struct {
				A int64 `toml:"a"`
			}{},
			want: "unknown key b",
		},
		{
			name: "invalid UTF-8",
			doc:  "a = \"\xff\"\n",
			v:    new(map[string]any),
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := decodeTOML([]byte(c.doc), c.v)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not contain %q", err, c.want)
			}
		})
	}
}

func TestDecodeTOMLAccepts(t *testing.T) {
	const backslash = "\\"
	cases := []struct {
		name string
		doc  string
		want map[string]any
	}{
		{
			name: "TOML v1.0.0",
			doc:  "a = \"x\"\nb = 1\n\n[c]\nd = true\n",
			want: map[string]any{
				"a": "x",
				"b": int64(1),
				"c": map[string]any{"d": true},
			},
		},
		{
			name: "literal string",
			doc:  `a = 'C:\e'` + "\n",
			want: map[string]any{"a": `C:\e`},
		},
		{
			name: "multiline literal string",
			doc:  `a = '''C:\e'''` + "\n",
			want: map[string]any{"a": `C:\e`},
		},
		{
			name: "escaped backslash",
			doc:  "a = \"" + backslash + backslash + "e\"\n",
			want: map[string]any{"a": `\e`},
		},
		{
			name: "escaped quote and backslash",
			doc:  "a = \"" + backslash + "\"" + backslash + backslash + "e\"\n",
			want: map[string]any{"a": `"\e`},
		},
		{
			name: "comment",
			doc:  `a = 1 # \e` + "\n",
			want: map[string]any{"a": int64(1)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			if err := decodeTOML([]byte(tc.doc), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// withBackslashes replaces every BS in s with a backslash.
func withBackslashes(s string) string {
	return strings.ReplaceAll(s, "BS", string(rune(92)))
}

func TestCheckTOMLSyntaxRejectsTOML11(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"newline in inline table", "a = {\nb = 1\n}\n"},
		{"newline in nested inline table", "a = { b = {\nc = 1 } }\n"},
		{"comment in inline table", "a = { b = 1 # c\n}\n"},
		{"trailing comma in inline table", "a = { b = 1, }\n"},
		{"trailing comma before space in inline table", "a = { b = 1 ,  }\n"},
		{"escape e", "a = \"BSe\"\n"},
		{"escape x", "a = \"BSx41\"\n"},
		{"escape x in multi-line string", "a = \"\"\"\nBSx41\"\"\"\n"},
		{"local time without seconds", "a = 07:32\n"},
		{"local date-time without seconds", "a = 1979-05-27T07:32\n"},
		{"offset date-time without seconds", "a = 1979-05-27 07:32Z\n"},
		{"time without seconds in array", "a = [07:32]\n"},
		{"offset date-time with T without seconds", "a = 1979-05-27T07:32Z\n"},
		{"escape e in multi-line string", "a = \"\"\"\nxBSey\n\"\"\"\n"},
		{"escape e in quoted key", "\"aBSeb\" = 1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkTOMLSyntax([]byte(withBackslashes(tc.doc))); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestCheckTOMLSyntaxAcceptsTOML10(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"array spanning lines in inline table", "a = { b = [\n1,\n2,\n] }\n"},
		{"trailing comma in array", "a = [1, 2,]\n"},
		{"inline table", "a = { b = 1, c = { d = 2 } }\n"},
		{"empty inline table", "a = {}\n"},
		{"local time", "a = 07:32:00\n"},
		{"offset date-time", "a = 1979-05-27T07:32:00.999-07:00\n"},
		{"braces, colons and escapes in strings", "a = \"{ 07:32 BSBSx BSBSe }\"\nb = '{BSx}'\nc = '''\n{\n07:32 }'''\n"},
		{"escapes of TOML v1.0.0", "a = \"BSbBStBSnBSfBSrBS\"BSBSBSu0041BSU00000041\"\n"},
		{"comment with braces and a time", "a = 1 # { 07:32,\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkTOMLSyntax([]byte(withBackslashes(tc.doc))); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTOMLConformance(t *testing.T) {
	if err := TOMLConformance(); err != nil {
		t.Fatal(err)
	}
}
