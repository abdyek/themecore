package themecore

import (
	"strconv"
	"strings"
	"testing"
)

func TestValuesAccepted(t *testing.T) {
	cases := []string{
		`a{width:calc(100% - var(--gap))}`,
		`a{color:rgb(0 0 0 / 50%)}`,
		`a{color:color-mix(in oklch, red, blue)}`,
		`a{background:linear-gradient(to right, red, blue)}`,
		`a{transform:translate(1px) rotate(45deg)}`,
		`a{filter:drop-shadow(0 0 1px red)}`,
		`a{clip-path:polygon(0 0, 100% 0)}`,
		`a{transition-timing-function:cubic-bezier(.1,.2,.3,.4)}`,
		`a{grid-template-columns:repeat(3, minmax(0, 1fr))}`,
		`a{--x: calc(1px + 2px)}`,
		`@font-face{font-family:x;src:local(Inter), url(fonts/a.woff2) format(\"woff2\") tech(variations)}`,
		`a{color:light-dark(white, black)}`,
		`a{width:clamp(1rem, 2vw, 3rem)}`,
		`@keyframes k{from{transform:scale(0)}}`,
		`a{animation-name:fade}`,
		`a{font-family:"umono-sans", sans-serif}`,
		`@layer base.components{a{color:red}}`,
		`@layer a, b.c;`,
		`@keyframes "fade-in"{to{opacity:1}}`,
		`a{counter-reset:item;counter-increment:item 2}`,
		`a{width:CALC(1px)}`,
	}
	for i, css := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if _, err := Transform([]byte(css)); err != nil {
				t.Fatalf("Transform(%q) returned error: %v", css, err)
			}
		})
	}
}

func TestValuesRejected(t *testing.T) {
	cases := []struct {
		css  string
		want string
	}{
		{`a{width:sibling-index()}`, "function sibling-index() is not allowed"},
		{`a{width:calc(1px * sibling-count())}`, "function sibling-count() is not allowed"},
		{`a{color:if(media(print): red)}`, "function if() is not allowed"},
		{`a{padding:env(safe-area-inset-top)}`, "function env() is not allowed"},
		{`a{background:element(#a)}`, "function element() is not allowed"},
		{`a{background:-webkit-gradient(linear)}`, "function -webkit-gradient() is not allowed"},
		{`a{background:-webkit-image-set("a.png" 1x)}`, "function -webkit-image-set() is not allowed"},
		{`a{background:paint(x)}`, "function paint() is not allowed"},
		{`a{top:anchor(top)}`, "function anchor() is not allowed"},
		{`a{animation-timeline:scroll(root)}`, "function scroll() is not allowed"},
		{`a{width:expression(1)}`, "function expression() is not allowed"},
		{`a{--x: env(x)}`, "function env() is not allowed"},
		{`@keyframes k{from{width:env(x)}}`, "function env() is not allowed"},
		{`@font-face{font-family:x;src:url(fonts/a.woff2) format(env(x))}`, "function env() is not allowed"},
		{`a{src:local(x)}`, "function local() is only allowed in the src descriptor of @font-face"},
		{`@font-face{font-family:local(x)}`, "function local() is only allowed in the src descriptor of @font-face"},
		{`a{behavior:url(x.htc)}`, "property behavior is not allowed"},
		{`a{-moz-binding:url(x)}`, "property -moz-binding is not allowed"},
		{`a{content:attr(id)}`, "attr() must name an attribute other than id"},
		{`a{content:attr(ID)}`, "attr() must name an attribute other than id"},
		{`a{content:attr(var(--a))}`, "attr() must name an attribute other than id"},
		{`a{--x: attr(id)}`, "attr() must name an attribute other than id"},
		{`a{--umono-err:red}`, "the umono- and compono- prefixes are reserved"},
		{`a{--Compono-x:red}`, "the umono- and compono- prefixes are reserved"},
		{`a{animation-name:umono-spin}`, "the umono- and compono- prefixes are reserved"},
		{`a{view-transition-name:compono-page}`, "the umono- and compono- prefixes are reserved"},
		{`a{color:var(--umono-x)}`, "the umono- and compono- prefixes are reserved"},
		{`a{anchor-name:--umono-a}`, "the umono- and compono- prefixes are reserved"},
		{`a{--x: umono-a}`, "the umono- and compono- prefixes are reserved"},
		{`@keyframes umono-spin{from{opacity:0}}`, "the umono- and compono- prefixes are reserved"},
		{`@keyframes "umono-spin"{from{opacity:0}}`, "the umono- and compono- prefixes are reserved"},
		{"@keyframes \\75mono-spin{from{opacity:0}}", "the umono- and compono- prefixes are reserved"},
		{`@font-face{font-family:"umono-sans";src:local(x)}`, "the umono- and compono- prefixes are reserved"},
		{`@font-face{font-family:umono-sans}`, "the umono- and compono- prefixes are reserved"},
		{`@layer umono-shell;`, "the umono- and compono- prefixes are reserved"},
		{`@layer umono-shell{a{color:red}}`, "the umono- and compono- prefixes are reserved"},
		{`@layer base, compono-x.y;`, "the umono- and compono- prefixes are reserved"},
		{`@layer a..b;`, "invalid @layer name"},
		{`@layer 1x;`, "invalid @layer name"},
		{`@layer a b;`, "invalid @layer name"},
		{`@keyframes a b{from{opacity:0}}`, "invalid @keyframes name"},
		{`@keyframes 1{from{opacity:0}}`, "invalid @keyframes name"},
	}
	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := Transform([]byte(tc.css))
			if err == nil {
				t.Fatalf("Transform(%q) returned nil error, want containing %q", tc.css, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Transform(%q) error = %q, want containing %q", tc.css, err.Error(), tc.want)
			}
		})
	}
}

func TestValuesVerify(t *testing.T) {
	css := []byte(`@scope (umono-root){a{width:env(x)}}`)
	want := "function env() is not allowed"
	if err := Verify(css); err == nil {
		t.Fatalf("Verify(%q) returned nil error, want containing %q", css, want)
	} else if !strings.Contains(err.Error(), want) {
		t.Fatalf("Verify(%q) error = %q, want containing %q", css, err.Error(), want)
	}
}
