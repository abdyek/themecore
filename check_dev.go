package themecore

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// DevResult is the outcome of CheckDev. The errors of one check are
// returned here and do not stop dev mode; Umono handles each on its own.
type DevResult struct {
	Contract Contract
	// Floor is DetectFeatureFloor of the dev package.
	Floor    int
	Evidence []Signal
	// CSS is the Transform output of assets/theme.css, nil if CSSErr is set.
	// Umono answers the theme.css request with an error status then.
	CSS    []byte
	CSSErr error
	// Attributes is the compiled attributes.toml, an empty table if there is
	// none or AttributesErr is set. Pages are rendered without injection then.
	Attributes    AttributeTable
	AttributesErr error
	// AssetErrs maps the path of every file under assets/ that breaks the
	// rules to its error. Umono answers its request with 404.
	AssetErrs map[string]error
}

// CheckDev runs the checks of activation on the dev directory dir of
// Umono's dev mode, themes/dev/dev/0.0.0/, with the privileges of dev/dev,
// which exist only here. root is the theme project, UMONO_THEME_DEV. There
// is no checksums.json; assets/, attributes.toml and contract.toml may be
// symlinks into root; theme.css goes through Transform instead of Verify;
// built.toml has the id dev/dev and the version 0.0.0. The other checks run
// as in CheckPackage. The package limits and content, built.toml,
// contract.toml, LICENSE and the feature floor are fatal: the returned
// error. CheckDev keeps no cache; Umono calls it on every request.
func CheckDev(dir, root string) (DevResult, error) {
	p, assetErrs, err := readDev(dir, root)
	if err != nil {
		return DevResult{}, err
	}
	for _, name := range []string{"built.toml", "contract.toml"} {
		if _, ok := p.Files[name]; !ok {
			return DevResult{}, fmt.Errorf("package: %s: missing", name)
		}
	}
	built, err := parseBuilt(p.Files["built.toml"], true)
	if err != nil {
		return DevResult{}, err
	}
	c, err := ParseContract(p.Files["contract.toml"])
	if err != nil {
		return DevResult{}, err
	}
	if err := built.checkContract(c.ID, c.Version); err != nil {
		return DevResult{}, err
	}

	r := DevResult{Contract: c}
	for _, name := range slices.Sorted(maps.Keys(p.Files)) {
		if !strings.HasPrefix(name, "assets/") || name == themeCSS {
			continue
		}
		if err := checkAsset(name, p.Files[name]); err != nil {
			if assetErrs == nil {
				assetErrs = map[string]error{}
			}
			assetErrs[name] = err
			delete(p.Files, name)
		}
	}
	r.AssetErrs = assetErrs

	if src, ok := p.Files[themeCSS]; !ok {
		r.CSSErr = fmt.Errorf("package: %s: missing", themeCSS)
	} else if css, err := Transform(src); err != nil {
		r.CSSErr = err
		delete(p.Files, themeCSS)
	} else {
		p.Files[themeCSS] = css
		if err := checkAssetURLs(p); err != nil {
			r.CSSErr = err
			delete(p.Files, themeCSS)
		} else {
			r.CSS = css
		}
	}

	if b, ok := p.Files["attributes.toml"]; ok {
		if r.Attributes, r.AttributesErr = CompileAttributes(b, c); r.AttributesErr != nil {
			delete(p.Files, "attributes.toml")
		}
	}
	if err := checkLicense(p); err != nil {
		return DevResult{}, err
	}
	floor, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		return DevResult{}, err
	}
	if built.ThemeAPIVersion < floor {
		return DevResult{}, fmt.Errorf("built.toml: theme-api-version: %d is lower than %d, the version of the features the package uses", built.ThemeAPIVersion, floor)
	}
	r.Floor, r.Evidence = floor, evidence
	return r, nil
}
