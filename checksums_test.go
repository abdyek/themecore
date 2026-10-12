package themecore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func withChecksums(t *testing.T, files map[string][]byte) Package {
	t.Helper()
	p := Package{Files: make(map[string][]byte, len(files))}
	for k, v := range files {
		p.Files[k] = v
	}
	out, err := Checksums(p)
	if err != nil {
		t.Fatalf("Checksums: %v", err)
	}
	p.Files["checksums.json"] = out
	return p
}

func TestChecksumsFormat(t *testing.T) {
	p := Package{Files: map[string][]byte{
		"LICENSE":          []byte("l"),
		"built.toml":       []byte("b"),
		"assets/theme.css": []byte("c"),
		"checksums.json":   []byte("old"),
	}}
	out, err := Checksums(p)
	if err != nil {
		t.Fatalf("Checksums: %v", err)
	}
	want := fmt.Sprintf("{\n  \"algorithm\": \"sha256\",\n  \"files\": {\n    \"LICENSE\": \"%s\",\n    \"assets/theme.css\": \"%s\",\n    \"built.toml\": \"%s\"\n  }\n}\n",
		hashOf("l"), hashOf("c"), hashOf("b"))
	if string(out) != want {
		t.Errorf("output mismatch:\ngot:  %q\nwant: %q", string(out), want)
	}
}

func TestVerifyChecksums(t *testing.T) {
	p := withChecksums(t, map[string][]byte{
		"built.toml":       []byte("b"),
		"contract.toml":    []byte("k"),
		"assets/theme.css": []byte("c"),
	})
	if err := VerifyChecksums(p); err != nil {
		t.Fatalf("VerifyChecksums: %v", err)
	}
	compact := fmt.Sprintf(`{"files":{"assets/theme.css":"%s","built.toml":"%s","contract.toml":"%s"},"algorithm":"sha256"}`,
		hashOf("c"), hashOf("b"), hashOf("k"))
	p.Files["checksums.json"] = []byte(compact)
	if err := VerifyChecksums(p); err != nil {
		t.Fatalf("VerifyChecksums (compact): %v", err)
	}
}

func TestVerifyChecksumsRejects(t *testing.T) {
	cases := []struct {
		name   string
		change func(p *Package)
		want   string
	}{
		{
			name: "missing",
			change: func(p *Package) {
				delete(p.Files, "checksums.json")
			},
			want: "package: checksums.json: missing",
		},
		{
			name: "not json",
			change: func(p *Package) {
				p.Files["checksums.json"] = []byte("{")
			},
			want: "package: checksums.json: not valid JSON",
		},
		{
			name: "trailing data",
			change: func(p *Package) {
				original := p.Files["checksums.json"]
				combined := make([]byte, 0, len(original)+2)
				combined = append(combined, original...)
				combined = append(combined, []byte("{}")...)
				p.Files["checksums.json"] = combined
			},
			want: "package: checksums.json: not valid JSON",
		},
		{
			name: "algorithm sha1",
			change: func(p *Package) {
				p.Files["checksums.json"] = []byte(`{"algorithm":"sha1","files":{}}`)
			},
			want: "package: checksums.json: algorithm must be sha256",
		},
		{
			name: "algorithm missing",
			change: func(p *Package) {
				p.Files["checksums.json"] = []byte(`{"files":{}}`)
			},
			want: "package: checksums.json: algorithm must be sha256",
		},
		{
			name: "unknown key",
			change: func(p *Package) {
				p.Files["checksums.json"] = []byte(`{"algorithm":"sha256","files":{},"version":1}`)
			},
			want: `package: checksums.json: unknown key "version"`,
		},
		{
			name: "file not listed",
			change: func(p *Package) {
				p.Files["assets/a.png"] = []byte("p")
			},
			want: "package: assets/a.png: missing from checksums.json",
		},
		{
			name: "listed file not in package",
			change: func(p *Package) {
				delete(p.Files, "assets/theme.css")
			},
			want: "package: checksums.json: assets/theme.css: not in the package",
		},
		{
			name: "lists itself",
			change: func(p *Package) {
				p.Files["checksums.json"] = []byte(fmt.Sprintf(
					`{"algorithm":"sha256","files":{"checksums.json":"%s","built.toml":"%s","assets/theme.css":"%s"}}`,
					hashOf("x"), hashOf("b"), hashOf("c")))
			},
			want: "package: checksums.json: lists itself",
		},
		{
			name: "changed file",
			change: func(p *Package) {
				p.Files["built.toml"] = []byte("B")
			},
			want: "package: built.toml: checksum does not match",
		},
		{
			name: "upper case hash",
			change: func(p *Package) {
				h := strings.ToUpper(hashOf("b"))
				p.Files["checksums.json"] = []byte(fmt.Sprintf(
					`{"algorithm":"sha256","files":{"built.toml":"%s","assets/theme.css":"%s"}}`,
					h, hashOf("c")))
			},
			want: "package: built.toml: checksum does not match",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := withChecksums(t, map[string][]byte{
				"built.toml":       []byte("b"),
				"assets/theme.css": []byte("c"),
			})
			tc.change(&p)
			err := VerifyChecksums(p)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
