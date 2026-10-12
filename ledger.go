package themecore

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
