package themecore

import "strings"

// The capability ledger. Rows are only appended: an existing row is never
// edited or deleted, and an ID is never reused. Extending the Theme API is a
// Feature row, narrowing it is a Break row.

// DetectorFunc reports whether a package uses a feature. where is the first
// place in the package that shows it, for example "assets/icons/cart.svg".
// Detectors look at the package itself: the canonical CSS and the package's
// own contract.toml, never the author's sources.
type DetectorFunc func(p Package) (where string, found bool)

// Feature is a capability of the Theme API and the version that introduced
// it. Every feature has a detector.
type Feature struct {
	ID           string
	IntroducedIn int
	Detect       DetectorFunc
}

// Break is a narrowing of the Theme API. It raises both ThemeAPICurrent and
// ThemeAPIMin to At.
type Break struct {
	At     int
	Reason string
}

// Features are the feature rows of the ledger.
var Features = []Feature{
	{ID: "pkg.attributes", IntroducedIn: 1, Detect: detectFile("attributes.toml")},
	{ID: "attr.class", IntroducedIn: 1, Detect: detectAddress(hasClass)},
	{ID: "attr.sizes", IntroducedIn: 1, Detect: detectAddress(hasSizes)},
	{ID: "attr.address.layout", IntroducedIn: 1, Detect: detectAddress(layoutAddress)},
	{ID: "attr.address.layout-builtin", IntroducedIn: 1, Detect: detectAddress(layoutBuiltinAddress)},
	{ID: "attr.address.layout-component-builtin", IntroducedIn: 1, Detect: detectAddress(layoutComponentBuiltinAddress)},
	{ID: "pkg.license", IntroducedIn: 1, Detect: detectFile("LICENSE")},
	{ID: "assets.woff2", IntroducedIn: 1, Detect: detectAsset(".woff2")},
	{ID: "assets.woff", IntroducedIn: 1, Detect: detectAsset(".woff")},
	{ID: "assets.png", IntroducedIn: 1, Detect: detectAsset(".png")},
	{ID: "assets.jpeg", IntroducedIn: 1, Detect: detectAsset(".jpeg")},
	{ID: "assets.jpg", IntroducedIn: 1, Detect: detectAsset(".jpg")},
	{ID: "assets.webp", IntroducedIn: 1, Detect: detectAsset(".webp")},
	{ID: "assets.avif", IntroducedIn: 1, Detect: detectAsset(".avif")},
	{ID: "assets.gif", IntroducedIn: 1, Detect: detectAsset(".gif")},
	{ID: "css.at.media", IntroducedIn: 1, Detect: detectAtRule("media")},
	{ID: "css.at.supports", IntroducedIn: 1, Detect: detectAtRule("supports")},
	{ID: "css.at.layer", IntroducedIn: 1, Detect: detectAtRule("layer")},
	{ID: "css.at.font-face", IntroducedIn: 1, Detect: detectAtRule("font-face")},
	{ID: "css.at.keyframes", IntroducedIn: 1, Detect: detectAtRule("keyframes")},
	{ID: "compono.builtin.link", IntroducedIn: 1, Detect: detectBuiltin("LINK")},
	{ID: "compono.builtin.image", IntroducedIn: 1, Detect: detectBuiltin("IMAGE")},
	{ID: "compono.builtin.nav", IntroducedIn: 1, Detect: detectBuiltin("NAV")},
	{ID: "compono.builtin.list", IntroducedIn: 1, Detect: detectBuiltin("LIST")},
	{ID: "compono.builtin.details", IntroducedIn: 1, Detect: detectBuiltin("DETAILS")},
	{ID: "compono.builtin.header", IntroducedIn: 1, Detect: detectBuiltin("HEADER")},
	{ID: "compono.builtin.main", IntroducedIn: 1, Detect: detectBuiltin("MAIN")},
	{ID: "compono.builtin.footer", IntroducedIn: 1, Detect: detectBuiltin("FOOTER")},
	{ID: "compono.builtin.article", IntroducedIn: 1, Detect: detectBuiltin("ARTICLE")},
	{ID: "compono.builtin.aside", IntroducedIn: 1, Detect: detectBuiltin("ASIDE")},
	{ID: "compono.markdown.heading", IntroducedIn: 1, Detect: detectMarkdown("heading")},
	{ID: "compono.markdown.p", IntroducedIn: 1, Detect: detectMarkdown("p")},
	{ID: "compono.markdown.br", IntroducedIn: 1, Detect: detectMarkdown("br")},
	{ID: "compono.markdown.em", IntroducedIn: 1, Detect: detectMarkdown("em")},
	{ID: "compono.markdown.strong", IntroducedIn: 1, Detect: detectMarkdown("strong")},
	{ID: "compono.markdown.link", IntroducedIn: 1, Detect: detectMarkdown("link")},
	{ID: "compono.markdown.inline-code", IntroducedIn: 1, Detect: detectMarkdown("inline-code")},
	{ID: "compono.markdown.code-block", IntroducedIn: 1, Detect: detectMarkdown("code-block")},
	{ID: "compono.markdown.list", IntroducedIn: 1, Detect: detectMarkdown("list")},
	{ID: "compono.markdown.blockquote", IntroducedIn: 1, Detect: detectMarkdown("blockquote")},
	{ID: "compono.markdown.hr", IntroducedIn: 1, Detect: detectMarkdown("hr")},
	{ID: "compono.markdown.table", IntroducedIn: 1, Detect: detectMarkdown("table")},
}

// Breaks are the break rows of the ledger.
var Breaks = []Break{
	{At: 1, Reason: "First Theme API version"},
}

// ThemeAPICurrent is the newest Theme API version: the largest IntroducedIn
// of Features and At of Breaks.
func ThemeAPICurrent() int { return themeAPICurrent(Features, Breaks) }

// ThemeAPIMin is the oldest Theme API version still accepted: the largest At
// of Breaks.
func ThemeAPIMin() int { return themeAPIMin(Breaks) }

// themeAPICurrent returns the max of IntroducedIn and At across both slices (0 if both are empty).
func themeAPICurrent(features []Feature, breaks []Break) int {
	max := 0
	for _, f := range features {
		if f.IntroducedIn > max {
			max = f.IntroducedIn
		}
	}
	for _, b := range breaks {
		if b.At > max {
			max = b.At
		}
	}
	return max
}

// themeAPIMin returns the largest At in breaks (0 if breaks is empty).
func themeAPIMin(breaks []Break) int {
	max := 0
	for _, b := range breaks {
		if b.At > max {
			max = b.At
		}
	}
	return max
}

// Builtins returns the names of the built-ins open to themes, from the
// compono.builtin.* rows of Features in ledger order: the last segment of
// the ID in upper case, with "-" as "_". Address validation, the detectors
// and the undefined call rule look only at this set, not at Compono's list.
// It returns a new slice on every call.
func Builtins() []string {
	var names []string
	for _, f := range Features {
		if rest, ok := strings.CutPrefix(f.ID, "compono.builtin."); ok {
			names = append(names, strings.ToUpper(strings.ReplaceAll(rest, "-", "_")))
		}
	}
	return names
}

// MarkdownElements returns the names of the Markdown elements open to
// themes, from the compono.markdown.* rows of Features in ledger order: the
// last segment of the ID, the name of the element's spec. It returns a new
// slice on every call.
func MarkdownElements() []string {
	var names []string
	for _, f := range Features {
		if rest, ok := strings.CutPrefix(f.ID, "compono.markdown."); ok {
			names = append(names, rest)
		}
	}
	return names
}
