package themecore

import "fmt"

// Result is the outcome of CheckPackage.
type Result struct {
	Contract Contract
	// Attributes is the compiled attributes.toml, an empty table if the
	// package has none.
	Attributes AttributeTable
	// Floor is the largest version of the features the package uses, 0 if
	// it uses none. Evidence holds one signal per feature it uses.
	Floor    int
	Evidence []Signal
}

// requiredPackageFiles are the files every package has.
var requiredPackageFiles = []string{"built.toml", "contract.toml", "checksums.json", themeCSS}

// CheckPackage runs the checks of install and activation on p, a package read
// with ReadZip or ReadDir, which check the package limits and content. The
// order is that of the static check table: the required files, built.toml,
// contract.toml and its match with built.toml, the assets, the canonical
// grammar of the CSS, the files its URLs point to, attributes.toml against
// the contract, LICENSE and, last, the feature floor: a theme-api-version
// lower than the features the package uses is rejected. checksums.json and
// the Theme API range are checked by the caller before.
func CheckPackage(p Package) (Result, error) {
	for _, name := range requiredPackageFiles {
		if _, ok := p.Files[name]; !ok {
			return Result{}, fmt.Errorf("package: %s: missing", name)
		}
	}
	built, err := ParseBuilt(p.Files["built.toml"])
	if err != nil {
		return Result{}, err
	}
	c, err := ParseContract(p.Files["contract.toml"])
	if err != nil {
		return Result{}, err
	}
	if err := built.checkContract(c.ID, c.Version); err != nil {
		return Result{}, err
	}
	if err := checkAssets(p); err != nil {
		return Result{}, err
	}
	if err := verify(p.Files[themeCSS]); err != nil {
		return Result{}, err
	}
	if err := checkAssetURLs(p); err != nil {
		return Result{}, err
	}
	var attributes AttributeTable
	if b, ok := p.Files["attributes.toml"]; ok {
		if attributes, err = CompileAttributes(b, c); err != nil {
			return Result{}, err
		}
	}
	if err := checkLicense(p); err != nil {
		return Result{}, err
	}
	floor, evidence, err := detectFeatureFloor(p)
	if err != nil {
		return Result{}, err
	}
	if built.ThemeAPIVersion < floor {
		return Result{}, fmt.Errorf("built.toml: theme-api-version: %d is lower than %d, the version of the features the package uses", built.ThemeAPIVersion, floor)
	}
	return Result{Contract: c, Attributes: attributes, Floor: floor, Evidence: evidence}, nil
}

// BuildReport is the outcome of CheckSource.
type BuildReport struct {
	// Version is theme-api-version: the larger of the version of the
	// features the package uses and ThemeAPIMin. The CLI writes it to
	// built.toml. Evidence holds one signal per feature the package uses.
	Version  int
	Evidence []Signal
	// Warnings are mistakes that are valid CSS but never match: root
	// classes used without :scope and addresses whose component the layout
	// never renders. They do not stop the build.
	Warnings []Warning
}

// CheckSource runs the checks of build and of the dev proxy on the
// development directory dir. contract is the file of the contract that
// theme.toml refers to, from the CLI's cache; themecore does not know the
// cache layout. The order is that of the static check table: the package
// limits and content, theme.toml, contract.toml, the assets, the CSS turned
// into the canonical CSS, the files the CSS URLs point to, attributes.toml
// against the contract and LICENSE. The returned package is what goes into
// the built file, the canonical CSS included; the CLI adds built.toml and
// then checksums.json.
func CheckSource(dir string, contract []byte) (Package, BuildReport, error) {
	p, manifest, err := readSource(dir, contract)
	if err != nil {
		return Package{}, BuildReport{}, err
	}
	if _, err := ParseManifest(manifest); err != nil {
		return Package{}, BuildReport{}, err
	}
	c, err := ParseContract(contract)
	if err != nil {
		return Package{}, BuildReport{}, err
	}
	if err := checkAssets(p); err != nil {
		return Package{}, BuildReport{}, err
	}
	canonical, err := transform(p.Files[themeCSS])
	if err != nil {
		return Package{}, BuildReport{}, err
	}
	p.Files[themeCSS] = canonical
	if err := checkAssetURLs(p); err != nil {
		return Package{}, BuildReport{}, err
	}
	if b, ok := p.Files["attributes.toml"]; ok {
		if _, err := CompileAttributes(b, c); err != nil {
			return Package{}, BuildReport{}, err
		}
	}
	if err := checkLicense(p); err != nil {
		return Package{}, BuildReport{}, err
	}
	version, evidence, err := detectThemeAPIVersion(p)
	if err != nil {
		return Package{}, BuildReport{}, err
	}
	return p, BuildReport{Version: version, Evidence: evidence, Warnings: lint(p)}, nil
}
