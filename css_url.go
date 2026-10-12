package themecore

import (
	"fmt"
	"path"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// valueURLs returns the URLs of toks in source order. The walk keeps a
// stack of enclosing function names: a function token pushes
// asciiLower(cssUnescape(name without "(")), a "(" pushes "", and a ")"
// pops.
//
// URLs come from three places:
//   - a css.URLToken: its data is like "url( a.png )"; the URL is the text
//     between the first "(" and the last ")", trimmed of CSS whitespace,
//     and CSS-unescaped;
//   - a function token named "url" or "src": the next non-whitespace token
//     must be a string, followed by optional whitespace and ")"; the URL
//     is the string's content. Otherwise valueURLs returns
//     "theme.css:<line>: invalid url()" at the function token's line;
//   - a string token whose innermost enclosing function is "image-set":
//     the URL is the string's content.
func valueURLs(toks []cssToken) ([]string, error) {
	var urls []string
	var stack []string
	for i, t := range toks {
		switch t.tt {
		case css.URLToken:
			data := string(t.data)
			open := strings.IndexByte(data, '(')
			close := strings.LastIndexByte(data, ')')
			if open < 0 || close < 0 || close <= open {
				return nil, fmt.Errorf("theme.css:%d: invalid url()", t.line)
			}
			inner := data[open+1 : close]
			inner = strings.TrimFunc(inner, func(r rune) bool {
				return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
			})
			// The tdewolff lexer also returns url("a.png") and
			// url('a.png') as a single URL token with the quotes
			// inside; drop matching outer quotes before unescaping.
			if len(inner) >= 2 && (inner[0] == '"' || inner[0] == '\'') && inner[len(inner)-1] == inner[0] {
				inner = inner[1 : len(inner)-1]
			}
			urls = append(urls, cssUnescape([]byte(inner)))
		case css.FunctionToken:
			data := t.data
			if len(data) > 0 && data[len(data)-1] == '(' {
				data = data[:len(data)-1]
			}
			name := asciiLower(cssUnescape(data))
			stack = append(stack, name)
			if name == "url" || name == "src" {
				j := i + 1
				for j < len(toks) && toks[j].tt == css.WhitespaceToken {
					j++
				}
				if j >= len(toks) || toks[j].tt != css.StringToken {
					return nil, fmt.Errorf("theme.css:%d: invalid url()", t.line)
				}
				urls = append(urls, stringContent(toks[j]))
				k := j + 1
				for k < len(toks) && toks[k].tt == css.WhitespaceToken {
					k++
				}
				if k >= len(toks) || toks[k].tt != css.RightParenthesisToken {
					return nil, fmt.Errorf("theme.css:%d: invalid url()", t.line)
				}
			}
		case css.LeftParenthesisToken:
			stack = append(stack, "")
		case css.RightParenthesisToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case css.StringToken:
			if len(stack) > 0 && stack[len(stack)-1] == "image-set" {
				urls = append(urls, stringContent(t))
			}
		}
	}
	return urls, nil
}

// checkURL reports whether u names a relative path to a file in assets/.
// The URL must be non-empty and not "theme.css", every segment of
// strings.Split(u, "/") must match segmentPattern and be at most
// maxSegmentLen bytes, and path.Ext(u) must be a key of assetTypes.
// Otherwise it returns a "must be a relative path to a file in assets/"
// error anchored at line.
func checkURL(u string, line int) error {
	if u == "" {
		return fmt.Errorf("theme.css:%d: url %s must be a relative path to a file in assets/", line, u)
	}
	if u == "theme.css" {
		return fmt.Errorf("theme.css:%d: url %s must be a relative path to a file in assets/", line, u)
	}
	for _, seg := range strings.Split(u, "/") {
		if len(seg) > maxSegmentLen {
			return fmt.Errorf("theme.css:%d: url %s must be a relative path to a file in assets/", line, u)
		}
		if !segmentPattern.MatchString(seg) {
			return fmt.Errorf("theme.css:%d: url %s must be a relative path to a file in assets/", line, u)
		}
	}
	if _, ok := assetTypes[path.Ext(u)]; !ok {
		return fmt.Errorf("theme.css:%d: url %s must be a relative path to a file in assets/", line, u)
	}
	return nil
}

// checkValueURLs returns the URLs of toks (valueURLs) and checks each with
// checkURL, anchored at line. The first failing check produces the
// returned error; on success it returns nil.
func checkValueURLs(toks []cssToken, line int) error {
	urls, err := valueURLs(toks)
	if err != nil {
		return err
	}
	for _, u := range urls {
		if err := checkURL(u, line); err != nil {
			return err
		}
	}
	return nil
}

// checkAssetURLs checks that every URL in the package's assets/theme.css
// names a file of the package. Transform and Verify only see the CSS, so
// they check the form of URLs; this check needs the package.
//
// If p.Files has no themeCSS, the package has no CSS and checkAssetURLs
// returns nil. Otherwise the CSS is parsed and walked recursively. URLs
// come from the declarations of style rules, @font-face, and @keyframes
// children, and from the prelude of @supports rules. For each URL in
// source order, if p.Files does not contain "assets/"+u, an error is
// returned. A parseCSS error is returned as is.
func checkAssetURLs(p Package) error {
	content, ok := p.Files[themeCSS]
	if !ok {
		return nil
	}
	rules, err := parseCSS(content)
	if err != nil {
		return err
	}
	return walkAssetURLs(rules, p)
}

// walkAssetURLs walks rules recursively and checks every URL against p.
// Style rules, @font-face, and @keyframes children expose URLs in their
// declarations; @supports exposes URLs in its prelude. @media, @layer, and
// @scope are not URL sources themselves; they are walked only to reach
// nested rules.
func walkAssetURLs(rules []cssRule, p Package) error {
	for _, r := range rules {
		switch {
		case r.at == "":
			if err := checkDeclAssetURLs(r.decls, p); err != nil {
				return err
			}
		case r.at == "font-face":
			if err := checkDeclAssetURLs(r.decls, p); err != nil {
				return err
			}
		case r.at == "keyframes":
			for _, child := range r.rules {
				if err := checkDeclAssetURLs(child.decls, p); err != nil {
					return err
				}
			}
		case r.at == "supports":
			if err := checkPreludeAssetURLs(r.prelude, p); err != nil {
				return err
			}
			if err := walkAssetURLs(r.rules, p); err != nil {
				return err
			}
		default:
			if err := walkAssetURLs(r.rules, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkDeclAssetURLs checks every URL of each declaration's value against p.
func checkDeclAssetURLs(decls []cssDecl, p Package) error {
	for _, d := range decls {
		if err := checkTokensAssetURLs(d.value, p); err != nil {
			return err
		}
	}
	return nil
}

// checkPreludeAssetURLs checks every URL of an @supports prelude against p.
func checkPreludeAssetURLs(prelude []cssToken, p Package) error {
	return checkTokensAssetURLs(prelude, p)
}

// checkTokensAssetURLs returns the URLs of toks (valueURLs) and reports
// the first URL that is not a key of p.Files with prefix "assets/".
func checkTokensAssetURLs(toks []cssToken, p Package) error {
	urls, err := valueURLs(toks)
	if err != nil {
		return err
	}
	for _, u := range urls {
		if _, ok := p.Files["assets/"+u]; !ok {
			return fmt.Errorf("package: assets/theme.css: url %s: file is not in the package", u)
		}
	}
	return nil
}
