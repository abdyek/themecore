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
	if len(featureRows) != 43 {
		t.Fatalf("expected 43 features, got %d", len(featureRows))
	}
	for i, id := range want {
		j := 20 + i
		if featureRows[j].ID != id {
			t.Errorf("featureRows[%d].ID = %q, want %q", j, featureRows[j].ID, id)
		}
		if featureRows[j].IntroducedIn != 1 {
			t.Errorf("featureRows[%d] %q IntroducedIn = %d, want 1", j, id, featureRows[j].IntroducedIn)
		}
	}
}

func TestBuiltins(t *testing.T) {
	want := []string{"LINK", "IMAGE", "NAV", "LIST", "DETAILS", "HEADER", "MAIN", "FOOTER", "ARTICLE", "ASIDE"}
	got := builtins()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("builtins() = %v, want %v", got, want)
	}
	got[0] = "CHANGED"
	if again := builtins(); !reflect.DeepEqual(again, want) {
		t.Errorf("builtins() after change = %v, want %v", again, want)
	}
}

func TestMarkdownElements(t *testing.T) {
	want := []string{"heading", "p", "br", "em", "strong", "link", "inline-code", "code-block", "list", "blockquote", "hr", "table"}
	got := markdownElements()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("markdownElements() = %v, want %v", got, want)
	}
	got[0] = "changed"
	if again := markdownElements(); !reflect.DeepEqual(again, want) {
		t.Errorf("markdownElements() after change = %v, want %v", again, want)
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
	for _, f := range featureRows {
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

func TestCurrentAutoDetector(t *testing.T) {
	if got := feature(t, "compono.link.current-auto").IntroducedIn; got != 2 {
		t.Errorf("compono.link.current-auto IntroducedIn = %d, want 2", got)
	}
	tests := []struct {
		name      string
		file      string
		wantWhere string
		wantFound bool
	}{
		{"direct", "direct.toml", "contract.toml:layouts[0].source", true},
		{"other-value", "other-value.toml", "", false},
		{"from-param", "from-param.toml", "", false},
		{"defined-name", "defined-name.toml", "", false},
		{"no-contract", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Package{Files: map[string][]byte{}}
			if tt.file != "" {
				b, err := os.ReadFile("testdata/current-auto/" + tt.file)
				if err != nil {
					t.Fatalf("read %s: %v", tt.file, err)
				}
				p.Files["contract.toml"] = b
			}
			where, found := feature(t, "compono.link.current-auto").Detect(p)
			if where != tt.wantWhere || found != tt.wantFound {
				t.Errorf("compono.link.current-auto: got (%q, %v), want (%q, %v)", where, found, tt.wantWhere, tt.wantFound)
			}
		})
	}
}

func TestDetectFeatureFloorUnreadableContract(t *testing.T) {
	p := Package{Files: map[string][]byte{"contract.toml": []byte("x = ")}}
	_, _, err := detectFeatureFloor(p)
	if err == nil {
		t.Fatalf("detectFeatureFloor: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "contract.toml") {
		t.Errorf("detectFeatureFloor: error %q does not contain %q", err.Error(), "contract.toml")
	}
}
