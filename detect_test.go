package themecore

import (
	"reflect"
	"strings"
	"testing"
)

func sameSignals(a, b []Signal) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func TestDetectFeatureFloor(t *testing.T) {
	cases := []struct {
		name     string
		files    map[string][]byte
		floor    int
		version  int
		evidence []Signal
	}{
		{
			name: "features",
			files: map[string][]byte{
				"attributes.toml": []byte(attrsDoc),
				"LICENSE":         []byte("MIT"),
				"assets/a.png":    []byte("x"),
				"built.toml":      []byte("x"),
			},
			floor:   1,
			version: 1,
			evidence: []Signal{
				{Feature: "pkg.attributes", IntroducedIn: 1, Where: "attributes.toml"},
				{Feature: "attr.class", IntroducedIn: 1, Where: "home"},
				{Feature: "attr.sizes", IntroducedIn: 1, Where: "post IMAGE"},
				{Feature: "attr.address.layout", IntroducedIn: 1, Where: "home"},
				{Feature: "attr.address.layout-builtin", IntroducedIn: 1, Where: "post IMAGE"},
				{Feature: "attr.address.layout-component-builtin", IntroducedIn: 1, Where: "post MAIN_MENU LINK"},
				{Feature: "pkg.license", IntroducedIn: 1, Where: "LICENSE"},
				{Feature: "assets.png", IntroducedIn: 1, Where: "assets/a.png"},
				{Feature: "compono.builtin.link", IntroducedIn: 1, Where: "post MAIN_MENU LINK"},
				{Feature: "compono.builtin.image", IntroducedIn: 1, Where: "post IMAGE"},
			},
		},
		{
			name:    "nothing",
			files:   map[string][]byte{"built.toml": []byte("x")},
			floor:   0,
			version: ThemeAPIMin(),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Package{Files: c.files}
			floor, evidence, err := detectFeatureFloor(p)
			if err != nil {
				t.Fatalf("detectFeatureFloor: unexpected error: %v", err)
			}
			if floor != c.floor {
				t.Errorf("detectFeatureFloor: floor: got %d want %d", floor, c.floor)
			}
			if !sameSignals(evidence, c.evidence) {
				t.Errorf("detectFeatureFloor: evidence mismatch:\ngot  %#v\nwant %#v", evidence, c.evidence)
			}
			version, versionEvidence, err := detectThemeAPIVersion(p)
			if err != nil {
				t.Fatalf("detectThemeAPIVersion: unexpected error: %v", err)
			}
			if version != c.version {
				t.Errorf("detectThemeAPIVersion: version: got %d want %d", version, c.version)
			}
			if !sameSignals(versionEvidence, c.evidence) {
				t.Errorf("detectThemeAPIVersion: evidence mismatch:\ngot  %#v\nwant %#v", versionEvidence, c.evidence)
			}
		})
	}
}

func TestDetectFeatureFloorUnreadable(t *testing.T) {
	cases := []struct {
		file    string
		content string
	}{
		{file: "attributes.toml", content: "x = "},
		{file: "assets/theme.css", content: "a{"},
		{file: "contract.toml", content: "x = "},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			p := Package{Files: map[string][]byte{c.file: []byte(c.content)}}
			_, _, err := detectFeatureFloor(p)
			if err == nil {
				t.Fatalf("detectFeatureFloor: expected error, got nil")
			}
			if !strings.Contains(err.Error(), c.file) {
				t.Errorf("detectFeatureFloor: error %q does not contain %q", err.Error(), c.file)
			}
		})
	}
}

func TestDetectFeatures(t *testing.T) {
	always := func(where string) func(Package) (string, bool) {
		return func(Package) (string, bool) { return where, true }
	}
	never := func(Package) (string, bool) { return "", false }
	features := []featureRow{
		{ID: "a.one", IntroducedIn: 1, Detect: always("x")},
		{ID: "a.three", IntroducedIn: 3, Detect: always("y")},
		{ID: "a.two", IntroducedIn: 2, Detect: never},
	}
	floor, evidence := detectFeatures(Package{}, features)
	if floor != 3 {
		t.Errorf("detectFeatures: floor: got %d want 3", floor)
	}
	want := []Signal{
		{Feature: "a.one", IntroducedIn: 1, Where: "x"},
		{Feature: "a.three", IntroducedIn: 3, Where: "y"},
	}
	if !reflect.DeepEqual(evidence, want) {
		t.Errorf("detectFeatures: evidence mismatch:\ngot  %#v\nwant %#v", evidence, want)
	}
}
