package themecore

import (
	"reflect"
	"strings"
	"testing"
)

func TestDetectFeatureFloor(t *testing.T) {
	p := Package{
		Files: map[string][]byte{
			"attributes.toml": []byte(attrsDoc),
			"LICENSE":         []byte("MIT"),
			"assets/a.png":    []byte("x"),
			"built.toml":      []byte("x"),
		},
	}
	floor, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		t.Fatalf("DetectFeatureFloor: unexpected error: %v", err)
	}
	if floor != 1 {
		t.Errorf("DetectFeatureFloor: floor: got %d want 1", floor)
	}
	want := []Signal{
		{Feature: "pkg.attributes", IntroducedIn: 1, Where: "attributes.toml"},
		{Feature: "attr.class", IntroducedIn: 1, Where: "home"},
		{Feature: "attr.sizes", IntroducedIn: 1, Where: "post IMAGE"},
		{Feature: "attr.address.layout", IntroducedIn: 1, Where: "home"},
		{Feature: "attr.address.layout-builtin", IntroducedIn: 1, Where: "post IMAGE"},
		{Feature: "attr.address.layout-component-builtin", IntroducedIn: 1, Where: "post MAIN_MENU LINK"},
		{Feature: "pkg.license", IntroducedIn: 1, Where: "LICENSE"},
		{Feature: "assets.png", IntroducedIn: 1, Where: "assets/a.png"},
	}
	if !reflect.DeepEqual(evidence, want) {
		t.Errorf("DetectFeatureFloor: evidence mismatch:\ngot  %#v\nwant %#v", evidence, want)
	}
}

func TestDetectFeatureFloorNothing(t *testing.T) {
	p := Package{Files: map[string][]byte{"built.toml": []byte("x")}}
	floor, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		t.Fatalf("DetectFeatureFloor: unexpected error: %v", err)
	}
	if floor != 0 {
		t.Errorf("DetectFeatureFloor: floor: got %d want 0", floor)
	}
	if len(evidence) != 0 {
		t.Errorf("DetectFeatureFloor: evidence: got %#v want empty", evidence)
	}
}

func TestDetectFeatureFloorUnreadableAttributes(t *testing.T) {
	p := Package{Files: map[string][]byte{"attributes.toml": []byte("x = ")}}
	_, _, err := DetectFeatureFloor(p)
	if err == nil {
		t.Fatalf("DetectFeatureFloor: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "attributes.toml") {
		t.Errorf("DetectFeatureFloor: error %q does not contain %q", err.Error(), "attributes.toml")
	}
}

func TestDetectThemeAPIVersion(t *testing.T) {
	pFull := Package{
		Files: map[string][]byte{
			"attributes.toml": []byte(attrsDoc),
			"LICENSE":         []byte("MIT"),
			"assets/a.png":    []byte("x"),
			"built.toml":      []byte("x"),
		},
	}
	version, evidence, err := DetectThemeAPIVersion(pFull)
	if err != nil {
		t.Fatalf("DetectThemeAPIVersion: unexpected error: %v", err)
	}
	if version != 1 {
		t.Errorf("DetectThemeAPIVersion: version: got %d want 1", version)
	}
	_, wantEvidence, floorErr := DetectFeatureFloor(pFull)
	if floorErr != nil {
		t.Fatalf("DetectFeatureFloor: unexpected error: %v", floorErr)
	}
	if !reflect.DeepEqual(evidence, wantEvidence) {
		t.Errorf("DetectThemeAPIVersion: evidence mismatch:\ngot  %#v\nwant %#v", evidence, wantEvidence)
	}

	pMin := Package{Files: map[string][]byte{"built.toml": []byte("x")}}
	versionMin, evidenceMin, err := DetectThemeAPIVersion(pMin)
	if err != nil {
		t.Fatalf("DetectThemeAPIVersion: unexpected error: %v", err)
	}
	if versionMin != ThemeAPIMin() {
		t.Errorf("DetectThemeAPIVersion: version: got %d want %d", versionMin, ThemeAPIMin())
	}
	if len(evidenceMin) != 0 {
		t.Errorf("DetectThemeAPIVersion: evidence: got %#v want empty", evidenceMin)
	}
}

func TestDetectFeatures(t *testing.T) {
	always := func(where string) func(Package) (string, bool) {
		return func(Package) (string, bool) { return where, true }
	}
	never := func(Package) (string, bool) { return "", false }
	features := []Feature{
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
