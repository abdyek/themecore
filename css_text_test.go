package themecore

import (
	"strconv"
	"strings"
	"testing"
)

func TestGeneratedTextAccepted(t *testing.T) {
	cases := []string{
		`a::before{content:""}`,
		`a::before{content:none}`,
		`a::before{content:normal}`,
		`a::before{content:"·" / ""}`,
		`a::before{content:"→" / ""}`,
		`a::before{content:url(i/a.png) / ""}`,
		`a::before{content:linear-gradient(red, blue) / ""}`,
		`a::before{content:counter(n) ". " / counter(n)}`,
		`a::before{content:attr(data-lang)}`,
		`q::before{content:open-quote}`,
		`q{quotes:auto}`,
		`q{quotes:"«" "»"}`,
		`li{list-style-type:"→ "}`,
		`li{list-style:square inside}`,
		`li{list-style-type:disc}`,
		`li::marker{content:counters(item, ".") " " / ""}`,
		`a{--label:"Read more"}`,
		`a::before{content:"\e900" / ""}`,
	}
	for i, css := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if _, err := Transform([]byte(css)); err != nil {
				t.Fatalf("Transform(%q) error = %v, want nil", css, err)
			}
		})
	}
}

func TestGeneratedTextRejected(t *testing.T) {
	tests := []struct {
		css  string
		want string
	}{
		{`a::before{content:"Read more" / ""}`, "strings in content must not contain letters"},
		{`a::before{content:"\52 ead" / ""}`, "strings in content must not contain letters"},
		{`a::before{content:"→" / "Devamı"}`, "strings in content must not contain letters"},
		{`li::marker{content:counters(item, "a") / ""}`, "strings in content must not contain letters"},
		{`a::before{content:"·"}`, "content with a string or an image needs alternative text"},
		{`a::before{content:url(i/a.png)}`, "content with a string or an image needs alternative text"},
		{`a::before{content:linear-gradient(red, blue)}`, "content with a string or an image needs alternative text"},
		{`a::before{content:"→" / "·"}`, "the alternative text of content must not contain a non-empty string"},
		{`a::before{content:"·" /}`, "content with a string or an image needs alternative text"},
		{`a::before{content:var(--label) / ""}`, "var() is not allowed in content"},
		{`q{quotes:"Ä" "B"}`, "strings in quotes must not contain letters"},
		{`li{list-style-type:"Step "}`, "strings in list-style-type must not contain letters"},
		{`li{list-style:"a" inside}`, "strings in list-style must not contain letters"},
		{`li{list-style:var(--x)}`, "var() is not allowed in list-style"},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := Transform([]byte(tt.css))
			if err == nil {
				t.Fatalf("Transform(%q) error = nil, want error containing %q", tt.css, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Transform(%q) error = %v, want error containing %q", tt.css, err, tt.want)
			}
		})
	}
}

func TestGeneratedTextVerify(t *testing.T) {
	css := `@scope (umono-root){a::before{content:"x" / ""}}`
	want := "strings in content must not contain letters"
	if err := Verify([]byte(css)); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Verify(%q) error = %v, want error containing %q", css, err, want)
	}
}
