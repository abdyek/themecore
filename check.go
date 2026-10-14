package themecore

import "fmt"

// Result is the outcome of CheckPackage.
type Result struct {
	Contract Contract
	// Attributes is the compiled attributes.toml, an empty table if the
	// package has none.
	Attributes AttributeTable
	// Floor is DetectFeatureFloor of the package.
	Floor    int
	Evidence []Signal
}

// requiredPackageFiles are the files every package has.
var requiredPackageFiles = []string{"built.toml", "contract.toml", "checksums.json", themeCSS}

// CheckPackage runs the checks of install and activation on p, a package read
// with ReadZip or ReadDir, which check the package limits and content. The
// order is that of the static check table: the required files, built.toml,
// contract.toml and its match with built.toml, the assets, Verify of the
// canonical CSS, the files its URLs point to, attributes.toml against the
// contract, LICENSE and, last, the feature floor: a theme-api-version lower
// than the features the package uses is rejected. checksums.json and the
// Theme API range are checked by the caller before.
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
	if err := Verify(p.Files[themeCSS]); err != nil {
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
	floor, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		return Result{}, err
	}
	if built.ThemeAPIVersion < floor {
		return Result{}, fmt.Errorf("built.toml: theme-api-version: %d is lower than %d, the version of the features the package uses", built.ThemeAPIVersion, floor)
	}
	return Result{Contract: c, Attributes: attributes, Floor: floor, Evidence: evidence}, nil
}
