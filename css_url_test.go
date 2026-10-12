package themecore

import (
	"strconv"
	"strings"
	"testing"
)

func TestURLsAccepted(t *testing.T) {
	sheets := []string{
		`@font-face{font-family:x;src:url(fonts/inter.woff2) format("woff2")}`,
		`a{background:url("img/a.png")}`,
		`a{background:url( img/a.png )}`,
		`a{background:image-set("a.png" 1x, "a-2x.png" 2x)}`,
		`a{background:image-set(url(a.png) 1x)}`,
		`a{--bg:url(a.webp)}`,
		`a::after{content:url(i/a.png) / ""}`,
		`a{mask:url(m.gif)}`,
		`a{cursor:url(c.png), auto}`,
		`@supports (background: url(a.png)) {a{color:red}}`,
		`a{background:url(a.jpeg), url(b.jpg), url(c.avif)}`,
	}
	for i, css := range sheets {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if _, err := Transform([]byte(css)); err != nil {
				t.Fatalf("Transform: %v", err)
			}
		})
	}
}

func TestURLsRejected(t *testing.T) {
	sheets := []string{
		`a{background:url(https://cdn.example.com/a.png)}`,
		`a{background:url("data:image/svg+xml,x")}`,
		`a{background:url(blob:x)}`,
		`a{background:url(//cdn.example.com/a.png)}`,
		`a{background:url(/media/logo.png)}`,
		`a{background:url(../built.toml)}`,
		`a{background:url(fonts/../../contract.toml)}`,
		`a{background:url(./fonts/a.png)}`,
		`a{background:url(fonts/a.png?v=2)}`,
		`a{mask:url(#mask)}`,
		`a{background:url(fonts/a%2Epng)}`,
		`a{background:url()}`,
		`a{background:url("")}`,
		`a{background:url(theme.css)}`,
		`a{background:url(a.svg)}`,
		`a{background:url(Logo.png)}`,
		`@supports (background: url(https://x.example/a.png)) {a{color:red}}`,
		`a{--bg:url(/a.png)}`,
		`a{background:image-set("https://x.example/a.png" 1x)}`,
		`a{background:src("/a.png")}`,
		`@font-face{font-family:x;src:url(https://x.example/a.woff2)}`,
	}
	want := "must be a relative path to a file in assets/"
	for i, css := range sheets {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := Transform([]byte(css))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not contain %q", err.Error(), want)
			}
		})
	}
}

func TestURLsInvalidForm(t *testing.T) {
	sheets := []string{
		`a{background:url(var(--x))}`,
		`a{background:src(var(--x))}`,
	}
	wants := []string{
		"bad url",
		"invalid url()",
	}
	for i, css := range sheets {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := Transform([]byte(css))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), wants[i]) {
				t.Fatalf("error %q does not contain %q", err.Error(), wants[i])
			}
		})
	}
}

func TestURLsVerify(t *testing.T) {
	css := `@scope (umono-root){a{background:url(/a.png)}}`
	want := "must be a relative path to a file in assets/"
	if err := Verify([]byte(css)); err == nil {
		t.Fatalf("expected error, got nil")
	} else if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

func TestCheckAssetURLs(t *testing.T) {
	tests := []struct {
		name  string
		files map[string][]byte
		want  string
	}{
		{
			name: "present",
			files: map[string][]byte{
				"assets/theme.css":     []byte(`@font-face{font-family:x;src:url(fonts/a.woff2)}`),
				"assets/fonts/a.woff2": []byte("x"),
			},
			want: "",
		},
		{
			name: "missing",
			files: map[string][]byte{
				"assets/theme.css": []byte(`@scope (umono-root){a{background:url(img/b.png)}}`),
			},
			want: "package: assets/theme.css: url img/b.png: file is not in the package",
		},
		{
			name: "no css",
			files: map[string][]byte{
				"built.toml": []byte("x"),
			},
			want: "",
		},
		{
			name: "unparsable",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{"),
			},
			want: "theme.css:1: unclosed block",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkAssetURLs(Package{Files: tc.files})
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}
