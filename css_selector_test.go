package themecore

import (
	"strings"
	"testing"
)

func transformSelector(sel string) error {
	_, err := Transform([]byte(sel + "{color:red}"))
	return err
}

func TestSelectorsAccepted(t *testing.T) {
	selectors := []string{
		`h1`,
		`*`,
		`.a`,
		`a.b.c`,
		`[data-lang]`,
		`[data-lang="go"]`,
		`[lang|=en]`,
		`a[href^="https:"]`,
		`[class~=x i]`,
		`a:hover`,
		`a:focus-visible`,
		`A:HOVER`,
		`:scope`,
		`:scope.layout-post`,
		`:scope.layout-post h1`,
		`:scope > nav`,
		`:scope h1`,
		`:scope:empty`,
		`:scope:has(> p:first-child)`,
		`:scope:has(p)`,
		`h1, h2`,
		`article > p + p`,
		`p ~ p`,
		`:is(article, section) h1`,
		`:where(.a)`,
		`:not(.a, .b)`,
		`li:nth-child(2n+1)`,
		`li:nth-child(odd)`,
		`li:nth-child(even of .a)`,
		`li:nth-last-child(-n+3)`,
		`li:nth-of-type(2n + 1)`,
		`li:nth-child( 3 )`,
		`li:nth-child(-n)`,
		`p:lang(tr)`,
		`p:lang("tr", en)`,
		`p:dir(rtl)`,
		`details:open`,
		`p::before`,
		`p::first-line`,
		`li::marker`,
		`::selection`,
		`nav > details::details-content`,
		`nav > details > summary`,
		`a:any-link`,
		`p:first-child`,
		`p:only-of-type`,
		`:has(+ p)`,
		`.a:not(:first-child)`,
		`p:is(.a, .b):hover`,
		`.\31 0`,
		`a:link:visited`,
	}
	for _, sel := range selectors {
		t.Run(sel, func(t *testing.T) {
			if err := transformSelector(sel); err != nil {
				t.Fatalf("unexpected error for %q: %v", sel, err)
			}
		})
	}
}

func TestSelectorsRejected(t *testing.T) {
	cases := []struct {
		selector string
		want     string
	}{
		{`html`, "html is not allowed"},
		{`HTML`, "html is not allowed"},
		{`\68tml`, "html is not allowed"},
		{`body p`, "body is not allowed"},
		{`umono-root`, "umono-root is not allowed"},
		{`:is(html) p`, "html is not allowed"},
		{`:root`, ":root is not allowed"},
		{`:is(:root) p`, ":root is not allowed"},
		{`&`, "& is not allowed"},
		{`a &`, "& is not allowed"},
		{`#giris`, "id selectors are not allowed"},
		{`[id]`, "id selectors are not allowed"},
		{`[ID]`, "id selectors are not allowed"},
		{`[id^="b"]`, "id selectors are not allowed"},
		{`:not(#x)`, "id selectors are not allowed"},
		{`:has([id])`, "id selectors are not allowed"},
		{`:target`, "id selectors are not allowed"},
		{`:target-within`, "id selectors are not allowed"},
		{`li:nth-child(1 of #a)`, "id selectors are not allowed"},
		{`*|html`, "namespace prefixes are not allowed"},
		{`|body`, "namespace prefixes are not allowed"},
		{`[*|id]`, "namespace prefixes are not allowed"},
		{`svg|a`, "namespace prefixes are not allowed"},
		{`:before`, "use ::before"},
		{`:after`, "use ::after"},
		{`:first-line`, "use ::first-line"},
		{`:first-letter`, "use ::first-letter"},
		{`:defined`, "unknown pseudo-class :defined"},
		{`:state(x)`, "unknown pseudo-class :state()"},
		{`:yeni(#giris)`, "unknown pseudo-class :yeni()"},
		{`:checked`, "unknown pseudo-class :checked"},
		{`:host`, "unknown pseudo-class :host"},
		{`:hover()`, "unknown pseudo-class :hover()"},
		{`:is`, "unknown pseudo-class :is"},
		{`::-webkit-details-marker`, "unknown pseudo-element ::-webkit-details-marker"},
		{`:-moz-focusring`, "unknown pseudo-class :-moz-focusring"},
		{`::part(x)`, "unknown pseudo-element ::part()"},
		{`body :scope h1`, "body is not allowed"},
		{`p :scope`, ":scope is only allowed in the first compound selector"},
		{`:is(:scope) h1`, ":scope is only allowed in the first compound selector"},
		{`:has(:scope)`, ":scope is only allowed in the first compound selector"},
		{`:scope ~ footer`, "the combinator after :scope must be a space or >"},
		{`:scope + aside`, "the combinator after :scope must be a space or >"},
		{`> h1`, "a selector cannot start with a combinator"},
		{`+ p`, "a selector cannot start with a combinator"},
		{`:scope:first-child`, "sibling-dependent pseudo-classes are not allowed in the :scope compound"},
		{`:scope:not(:last-child)`, "sibling-dependent pseudo-classes are not allowed in the :scope compound"},
		{`:scope:nth-child(2)`, "sibling-dependent pseudo-classes are not allowed in the :scope compound"},
		{`:scope:is(:only-of-type)`, "sibling-dependent pseudo-classes are not allowed in the :scope compound"},
		{`:scope:has(+ aside)`, ":has() with + or ~ is not allowed in the :scope compound"},
		{`:scope:has(~ p)`, ":has() with + or ~ is not allowed in the :scope compound"},
		{`:is(nav a)`, "combinators are not allowed inside :is()"},
		{`:not(* *)`, "combinators are not allowed inside :not()"},
		{`:where(a > b)`, "combinators are not allowed inside :where()"},
		{`:scope > p:is(:first-child > *)`, "combinators are not allowed inside :is()"},
		{`p:not(:nth-child(2) > *)`, "combinators are not allowed inside :not()"},
		{`li:nth-child(1 of a b)`, "combinators are not allowed inside :nth-child()"},
		{`:is()`, "invalid selector"},
		{`:has()`, "invalid selector"},
		{`a,`, "invalid selector"},
		{`,a`, "invalid selector"},
		{`a,,b`, "invalid selector"},
		{`a || b`, "invalid selector"},
		{`a > > b`, "invalid selector"},
		{`p::before::after`, "pseudo-elements must come last"},
		{`p::before.a`, "pseudo-elements must come last"},
		{`p::before span`, "pseudo-elements must come last"},
		{`li:nth-child(2 n)`, "invalid argument of :nth-child()"},
		{`li:nth-child(foo)`, "invalid argument of :nth-child()"},
		{`li:nth-child()`, "invalid argument of :nth-child()"},
		{`li:nth-child(- n)`, "invalid argument of :nth-child()"},
		{`li:nth-of-type(2n of .a)`, "invalid argument of :nth-of-type()"},
		{`p:dir(up)`, "invalid argument of :dir()"},
		{`p:lang()`, "invalid argument of :lang()"},
		{`p:lang(1)`, "invalid argument of :lang()"},
		{`[x=]`, "invalid attribute selector"},
		{`[=x]`, "invalid attribute selector"},
		{`[x="a" q]`, "invalid attribute selector"},
		{`.`, "invalid selector"},
	}
	for _, tc := range cases {
		t.Run(tc.selector, func(t *testing.T) {
			err := transformSelector(tc.selector)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestSelectorsVerify(t *testing.T) {
	t.Run("html not allowed", func(t *testing.T) {
		err := Verify([]byte(`@scope (umono-root){html{color:red}}`))
		if err == nil {
			t.Fatalf("expected error containing %q, got nil", "html is not allowed")
		}
		if !strings.Contains(err.Error(), "html is not allowed") {
			t.Fatalf("expected error containing %q, got %q", "html is not allowed", err.Error())
		}
	})
	t.Run("scope and pseudo-element accepted", func(t *testing.T) {
		if err := Verify([]byte(`@scope (umono-root){:scope>nav,p::before{color:red}}`)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
