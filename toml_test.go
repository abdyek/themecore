package themecore

import (
	"reflect"
	"testing"
)

func TestDecodeTOMLAcceptsTOML10(t *testing.T) {
	doc := "a = \"x\"\nb = 1\n\n[c]\nd = true\n"
	var got map[string]any
	if err := decodeTOML([]byte(doc), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{
		"a": "x",
		"b": int64(1),
		"c": map[string]any{"d": true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDecodeTOMLRejectsTOML11(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{
			name: "newline in inline table",
			doc:  "a = {\nb = 1\n}\n",
		},
		{
			name: "trailing comma in inline table",
			doc:  "a = { b = 1, }\n",
		},
		{
			name: "escape e",
			doc:  `a = "\e"` + "\n",
		},
		{
			name: "escape x",
			doc:  `a = "\x41"` + "\n",
		},
		{
			name: "local time without seconds",
			doc:  "a = 07:32\n",
		},
		{
			name: "offset date-time without seconds",
			doc:  "a = 1979-05-27T07:32Z\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			if err := decodeTOML([]byte(tc.doc), &got); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestDecodeTOMLRejectsUnknownKeys(t *testing.T) {
	doc := "a = 1\nb = 2\n"
	var got struct {
		A int64 `toml:"a"`
	}
	if err := decodeTOML([]byte(doc), &got); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDecodeTOMLRejectsInvalidUTF8(t *testing.T) {
	input := []byte("a = \"\xff\"\n")
	var got map[string]any
	if err := decodeTOML(input, &got); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDecodeTOMLRejectsEscapeE(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{
			name: "multiline basic string",
			doc:  `a = """` + "\n" + `x\ey` + "\n" + `"""` + "\n",
		},
		{
			name: "quoted key",
			doc:  `"a\eb" = 1` + "\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			if err := decodeTOML([]byte(tc.doc), &got); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestDecodeTOMLAcceptsBackslashE(t *testing.T) {
	const backslash = "\\"
	cases := []struct {
		name string
		doc  string
		want map[string]any
	}{
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
