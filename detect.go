package themecore

import "fmt"

// Signal is the evidence that a package uses a feature: the feature's ID, the
// version that introduced it and the first place in the package that shows
// it.
type Signal struct {
	Feature      string // "assets.svg"
	IntroducedIn int    // 3
	Where        string // "assets/icons/cart.svg"
}

// DetectFeatureFloor is the first term of theme-api-version: the largest
// IntroducedIn of the features the package uses, or 0 if it uses none.
// evidence has one signal per detected feature, in ledger order. Install and
// activation call it to verify the version that built.toml declares.
func DetectFeatureFloor(p Package) (floor int, evidence []Signal, err error) {
	if b, ok := p.Files["attributes.toml"]; ok {
		if _, err := decodeAddresses(b); err != nil {
			return 0, nil, fmt.Errorf("attributes.toml: cannot be read for feature detection: %w", err)
		}
	}
	if b, ok := p.Files[themeCSS]; ok {
		if _, err := parseCSS(b); err != nil {
			return 0, nil, fmt.Errorf("assets/theme.css: cannot be read for feature detection: %w", err)
		}
	}
	floor, evidence = detectFeatures(p, Features)
	return floor, evidence, nil
}

// DetectThemeAPIVersion is theme-api-version: the larger of
// DetectFeatureFloor and ThemeAPIMin. evidence is DetectFeatureFloor's. Build
// calls it to declare the version.
func DetectThemeAPIVersion(p Package) (version int, evidence []Signal, err error) {
	floor, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		return 0, nil, err
	}
	version = floor
	if min := ThemeAPIMin(); min > version {
		version = min
	}
	return version, evidence, nil
}

// detectFeatures runs the detector of every feature on p.
func detectFeatures(p Package, features []Feature) (floor int, evidence []Signal) {
	for _, f := range features {
		where, found := f.Detect(p)
		if !found {
			continue
		}
		evidence = append(evidence, Signal{Feature: f.ID, IntroducedIn: f.IntroducedIn, Where: where})
		if f.IntroducedIn > floor {
			floor = f.IntroducedIn
		}
	}
	return floor, evidence
}
