package themecore

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLedgerComponoRows(t *testing.T) {
	want := []string{
		"compono.builtin.link",
		"compono.builtin.image",
		"compono.builtin.nav",
		"compono.builtin.list",
		"compono.builtin.details",
		"compono.builtin.header",
		"compono.builtin.main",
		"compono.builtin.footer",
		"compono.builtin.article",
		"compono.builtin.aside",
		"compono.markdown.heading",
		"compono.markdown.p",
		"compono.markdown.br",
		"compono.markdown.em",
		"compono.markdown.strong",
		"compono.markdown.link",
		"compono.markdown.inline-code",
		"compono.markdown.code-block",
		"compono.markdown.list",
		"compono.markdown.blockquote",
		"compono.markdown.hr",
		"compono.markdown.table",
	}
	if len(Features) != 42 {
		t.Fatalf("expected 42 features, got %d", len(Features))
	}
	for i, id := range want {
		j := 20 + i
		if Features[j].ID != id {
			t.Errorf("Features[%d].ID = %q, want %q", j, Features[j].ID, id)
		}
		if Features[j].IntroducedIn != 1 {
			t.Errorf("Features[%d] %q IntroducedIn = %d, want 1", j, id, Features[j].IntroducedIn)
		}
	}
}

func TestBuiltins(t *testing.T) {
	want := []string{"LINK", "IMAGE", "NAV", "LIST", "DETAILS", "HEADER", "MAIN", "FOOTER", "ARTICLE", "ASIDE"}
	got := Builtins()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Builtins() = %v, want %v", got, want)
	}
	got[0] = "CHANGED"
	if again := Builtins(); !reflect.DeepEqual(again, want) {
		t.Errorf("Builtins() after change = %v, want %v", again, want)
	}
}

func TestMarkdownElements(t *testing.T) {
	want := []string{"heading", "p", "br", "em", "strong", "link", "inline-code", "code-block", "list", "blockquote", "hr", "table"}
	got := MarkdownElements()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MarkdownElements() = %v, want %v", got, want)
	}
	got[0] = "changed"
	if again := MarkdownElements(); !reflect.DeepEqual(again, want) {
		t.Errorf("MarkdownElements() after change = %v, want %v", again, want)
	}
}

func readDetectorFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/detectors/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestComponoDetectors(t *testing.T) {
	p := Package{Files: map[string][]byte{
		"contract.toml":   readDetectorFixture(t, "contract.toml"),
		"attributes.toml": readDetectorFixture(t, "attributes.toml"),
	}}
	want := map[string]string{
		"compono.builtin.link":     "contract.toml:layouts[0].source:4",
		"compono.builtin.image":    "contract.toml:layouts[0].source:1",
		"compono.builtin.details":  "post DETAILS",
		"compono.builtin.main":     "contract.toml:components[0].source:3",
		"compono.markdown.heading": "contract.toml:layouts[0].source:2",
		"compono.markdown.p":       "contract.toml:layouts[0].source:9",
		"compono.markdown.br":      "contract.toml:components[1].source:1",
		"compono.markdown.strong":  "contract.toml:components[0].source:2",
	}
	for _, f := range Features {
		if !strings.HasPrefix(f.ID, "compono.") {
			continue
		}
		where, found := f.Detect(p)
		wantWhere, wantFound := want[f.ID]
		if found != wantFound || where != wantWhere {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", f.ID, where, found, wantWhere, wantFound)
		}
	}
}

func TestBuiltinDetectorWithoutContract(t *testing.T) {
	p := Package{Files: map[string][]byte{
		"attributes.toml": readDetectorFixture(t, "attributes.toml"),
	}}
	where, found := feature(t, "compono.builtin.aside").Detect(p)
	if !found || where != "post ASIDE" {
		t.Errorf("compono.builtin.aside: got (%q, %v), want (%q, true)", where, found, "post ASIDE")
	}
	if _, found := feature(t, "compono.markdown.p").Detect(p); found {
		t.Errorf("compono.markdown.p: found without contract.toml")
	}
}

func TestDetectFeatureFloorUnreadableContract(t *testing.T) {
	p := Package{Files: map[string][]byte{"contract.toml": []byte("x = ")}}
	_, _, err := DetectFeatureFloor(p)
	if err == nil {
		t.Fatalf("DetectFeatureFloor: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "contract.toml") {
		t.Errorf("DetectFeatureFloor: error %q does not contain %q", err.Error(), "contract.toml")
	}
}
