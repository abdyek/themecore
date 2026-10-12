package themecore

import (
	"fmt"

	"github.com/tdewolff/parse/v2/css"
)

// cssContext identifies where a cssRule is being walked during grammar
// validation: cssContextTop at the top level of a stylesheet,
// cssContextBlock inside a @media, @supports, or @layer block, and
// cssContextScope inside an @scope block.
type cssContext int

const (
	// cssContextTop is the top level of a stylesheet.
	cssContextTop cssContext = iota
	// cssContextBlock is inside a @media, @supports, or @layer block.
	cssContextBlock
	// cssContextScope is inside an @scope block.
	cssContextScope
)

// checkGrammar checks the structure of a stylesheet: the author grammar of
// Transform's input (themecore, Karar 5), or with canonical set, the
// canonical grammar of the package's CSS (Karar 9). The two differ only in
// where style rules and @scope may appear.
func checkGrammar(rules []cssRule, canonical bool) error {
	return checkGrammarIn(rules, canonical, cssContextTop)
}

// checkGrammarIn walks rules in ctx and returns the first grammar error
// found, or nil. See checkGrammar for the rules.
func checkGrammarIn(rules []cssRule, canonical bool, ctx cssContext) error {
	for _, r := range rules {
		if err := checkRule(r, canonical, ctx); err != nil {
			return err
		}
	}
	return nil
}

// checkRule validates r in context ctx and returns the first grammar
// error, or nil. The order of checks is: a style rule is rejected when
// canonical and ctx is not scope; any at-rule in scope is rejected; then
// the at-rule-specific checks apply.
func checkRule(r cssRule, canonical bool, ctx cssContext) error {
	if r.at == "" {
		if err := checkSelector(r.prelude, r.line); err != nil {
			return err
		}
		if canonical && ctx != cssContextScope {
			return fmt.Errorf("theme.css:%d: style rule outside @scope (umono-root)", r.line)
		}
		return nil
	}
	if ctx == cssContextScope {
		return fmt.Errorf("theme.css:%d: @scope may contain only style rules", r.line)
	}
	switch r.at {
	case "scope":
		if !canonical {
			return fmt.Errorf("theme.css:%d: @scope is not allowed", r.line)
		}
		if !r.block {
			return fmt.Errorf("theme.css:%d: @scope must have a block", r.line)
		}
		if err := checkScopePrelude(r.prelude, r.line); err != nil {
			return err
		}
		return checkGrammarIn(r.rules, canonical, cssContextScope)
	case "media", "supports":
		if !r.block {
			return fmt.Errorf("theme.css:%d: @%s must have a block", r.line, r.at)
		}
		return checkGrammarIn(r.rules, canonical, cssContextBlock)
	case "layer":
		if r.block {
			return checkGrammarIn(r.rules, canonical, cssContextBlock)
		}
		if ctx != cssContextTop {
			return fmt.Errorf("theme.css:%d: @layer statements are only allowed at the top level", r.line)
		}
		if len(trimSpace(r.prelude)) == 0 {
			return fmt.Errorf("theme.css:%d: @layer statement must name a layer", r.line)
		}
		return nil
	case "font-face":
		if ctx != cssContextTop {
			return fmt.Errorf("theme.css:%d: @font-face is only allowed at the top level", r.line)
		}
		if !r.block {
			return fmt.Errorf("theme.css:%d: @font-face must have a block", r.line)
		}
		if len(trimSpace(r.prelude)) > 0 {
			return fmt.Errorf("theme.css:%d: @font-face takes no prelude", r.line)
		}
		for _, d := range r.decls {
			if d.important {
				return fmt.Errorf("theme.css:%d: !important is not allowed in @font-face", r.line)
			}
		}
		return nil
	case "keyframes":
		if ctx != cssContextTop {
			return fmt.Errorf("theme.css:%d: @keyframes is only allowed at the top level", r.line)
		}
		if !r.block {
			return fmt.Errorf("theme.css:%d: @keyframes must have a block", r.line)
		}
		if len(trimSpace(r.prelude)) == 0 {
			return fmt.Errorf("theme.css:%d: @keyframes must have a name", r.line)
		}
		for _, child := range r.rules {
			if child.at != "" {
				return fmt.Errorf("theme.css:%d: keyframe selector must be from, to or a percentage", child.line)
			}
			if err := checkKeyframeSelector(child.prelude, child.line); err != nil {
				return err
			}
			for _, d := range child.decls {
				if d.important {
					return fmt.Errorf("theme.css:%d: !important is not allowed in @keyframes", child.line)
				}
			}
		}
		return nil
	}
	return nil
}

// checkScopePrelude validates the prelude of an @scope at-rule: ignoring
// whitespace tokens, it must be exactly "(", an ident whose data is
// "umono-root" without escapes, and ")".
func checkScopePrelude(prelude []cssToken, line int) error {
	var toks []cssToken
	for _, t := range prelude {
		if t.tt == css.WhitespaceToken {
			continue
		}
		toks = append(toks, t)
	}
	if len(toks) != 3 {
		return fmt.Errorf("theme.css:%d: @scope prelude must be (umono-root)", line)
	}
	if toks[0].tt != css.LeftParenthesisToken {
		return fmt.Errorf("theme.css:%d: @scope prelude must be (umono-root)", line)
	}
	if toks[1].tt != css.IdentToken || string(toks[1].data) != "umono-root" {
		return fmt.Errorf("theme.css:%d: @scope prelude must be (umono-root)", line)
	}
	if toks[2].tt != css.RightParenthesisToken {
		return fmt.Errorf("theme.css:%d: @scope prelude must be (umono-root)", line)
	}
	return nil
}

// checkKeyframeSelector validates the prelude of a style rule inside a
// @keyframes block: ignoring whitespace, it must be a non-empty
// comma-separated list whose each item is a single ident "from"/"to" or a
// single percentage token.
func checkKeyframeSelector(prelude []cssToken, line int) error {
	var toks []cssToken
	for _, t := range prelude {
		if t.tt == css.WhitespaceToken {
			continue
		}
		toks = append(toks, t)
	}
	if len(toks) == 0 {
		return fmt.Errorf("theme.css:%d: keyframe selector must be from, to or a percentage", line)
	}
	items := splitComma(toks)
	for _, item := range items {
		if len(item) != 1 {
			return fmt.Errorf("theme.css:%d: keyframe selector must be from, to or a percentage", line)
		}
		t := item[0]
		switch t.tt {
		case css.PercentageToken:
		case css.IdentToken:
			s := asciiLower(cssUnescape(t.data))
			if s != "from" && s != "to" {
				return fmt.Errorf("theme.css:%d: keyframe selector must be from, to or a percentage", line)
			}
		default:
			return fmt.Errorf("theme.css:%d: keyframe selector must be from, to or a percentage", line)
		}
	}
	return nil
}

// splitComma splits toks into a slice of slices at every CommaToken. The
// returned items preserve the original order. An empty item is preserved
// for adjacent commas and for a trailing comma.
func splitComma(toks []cssToken) [][]cssToken {
	var items [][]cssToken
	var current []cssToken
	for _, t := range toks {
		if t.tt == css.CommaToken {
			items = append(items, current)
			current = nil
			continue
		}
		current = append(current, t)
	}
	items = append(items, current)
	return items
}
