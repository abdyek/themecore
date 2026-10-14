package themecore

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/umono-cms/compono"
)

func fixtureContract(t *testing.T) Contract {
	t.Helper()
	b, err := os.ReadFile("testdata/attributes/contract.toml")
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	c, err := ParseContract(b)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	return c
}

// oneAddressDoc returns an attributes.toml with one address and one attribute.
func oneAddressDoc(address, attr, value string) []byte {
	q := string(rune(34))
	return []byte("schema = " + q + "https://umono.dev/schemas/theme-attributes/v1" + q + "\n\n[address." + q + address + q + "]\n" + attr + " = " + q + value + q + "\n")
}

func TestCompileAttributes(t *testing.T) {
	b, err := os.ReadFile("testdata/attributes/attributes.toml")
	if err != nil {
		t.Fatalf("read attributes: %v", err)
	}
	table, err := CompileAttributes(b, fixtureContract(t))
	if err != nil {
		t.Fatalf("CompileAttributes: %v", err)
	}

	if got, want := table.Root("post"), map[string]string{"class": "layout-post"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Root(post) = %v, want %v", got, want)
	}
	if got := table.Root("home"); got != nil {
		t.Errorf("Root(home) = %v, want nil", got)
	}

	layoutFrame := compono.Frame{Name: "UMONO_LAYOUT", Kind: compono.FrameGlobal, ScopePath: []string{}}
	component := func(name string) compono.Frame {
		return compono.Frame{Name: name, Kind: compono.FrameGlobal, ScopePath: []string{"UMONO_LAYOUT"}}
	}
	cover := map[string]string{"class": "cover w-1/2 bg-[#fff]", "sizes": "(min-width: 48rem) 33vw, 100vw"}
	cases := []struct {
		name    string
		builtin string
		chain   []compono.Frame
		want    map[string]string
	}{
		{"layout scope", "IMAGE", []compono.Frame{layoutFrame}, cover},
		{"exposed component scope", "LINK", []compono.Frame{layoutFrame, component("MAIN_MENU")}, map[string]string{"class": "main-menu-link"}},
		{"no address in the scope", "IMAGE", []compono.Frame{layoutFrame, component("MAIN_MENU")}, nil},
		{"innermost exposed component", "IMAGE", []compono.Frame{layoutFrame, component("MAIN_MENU"), component("POST_BODY")}, map[string]string{"sizes": "100vw"}},
		{"component that is not exposed", "IMAGE", []compono.Frame{layoutFrame, component("HIDDEN")}, cover},
		{"local with the name of an exposed component", "LINK", []compono.Frame{layoutFrame, {Name: "MAIN_MENU", Kind: compono.FrameLocal}}, nil},
		{"site global with the name of an exposed component", "LINK", []compono.Frame{{Name: "MAIN_MENU", Kind: compono.FrameGlobal, ScopePath: []string{}}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := table.Lookup("post", tc.builtin, tc.chain); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Lookup = %v, want %v", got, tc.want)
			}
		})
	}

	got := table.Lookup("post", "IMAGE", nil)
	got["class"] = "changed"
	if again := table.Lookup("post", "IMAGE", nil); !reflect.DeepEqual(again, cover) {
		t.Errorf("Lookup after change = %v, want %v", again, cover)
	}
}

func TestCompileAttributesRejects(t *testing.T) {
	cases := []struct {
		name    string
		address string
		attr    string
		value   string
		want    string
	}{
		{"double space", "post  IMAGE", "class", "a", "parts must be separated by a single space"},
		{"leading space", " post", "class", "a", "parts must be separated by a single space"},
		{"too many parts", "post MAIN_MENU LINK IMAGE", "class", "a", "must be layout, layout BUILTIN or layout COMPONENT BUILTIN"},
		{"unknown layout", "blog", "class", "a", "blog is not a layout of the contract"},
		{"layout COMPONENT", "post MAIN_MENU", "class", "a", "MAIN_MENU is an exposed component, which has no injection point"},
		{"component that is not exposed", "post HIDDEN LINK", "class", "a", "HIDDEN is not an exposed component of the contract"},
		{"component named like a built-in", "post NAV", "class", "a", "NAV is not a built-in"},
		{"local named like a built-in", "home LIST", "class", "a", "LIST is not a built-in"},
		{"unknown built-in", "post TABLE", "class", "a", "TABLE is not a built-in"},
		{"attribute not allowed", "post", "id", "a", "id: not an allowed attribute"},
		{"sizes not on IMAGE", "post LINK", "sizes", "100vw", "sizes: only for an address that ends with IMAGE"},
		{"sizes grammar", "post IMAGE", "sizes", "auto", "sizes: not a valid source size list"},
		{"class empty", "post", "class", "", "class: must not be empty"},
		{"class double space", "post", "class", "a  b", "class: tokens must be separated by a single space"},
		{"class trailing space", "post", "class", "a ", "class: tokens must be separated by a single space"},
		{"class tab", "post", "class", "a" + string(rune(9)) + "b", "class: tokens must be printable ASCII"},
		{"class non-ASCII", "post", "class", "ğ", "class: tokens must be printable ASCII"},
		{"class duplicate", "post", "class", "a b a", "class: duplicate token a"},
		{"class umono prefix", "post", "class", "Umono-x", "class: token Umono-x uses a reserved prefix"},
		{"class compono prefix", "post", "class", "compono-y", "class: token compono-y uses a reserved prefix"},
		{"class too long", "post", "class", strings.Repeat("a", 1025), "class: longer than 1024 bytes"},
	}
	c := fixtureContract(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileAttributes(oneAddressDoc(tc.address, tc.attr, tc.value), c)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.HasPrefix(err.Error(), "attributes.toml: ") || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not start with %q and contain %q", err.Error(), "attributes.toml: ", tc.want)
			}
		})
	}
}

func TestCompileAttributesRejectsEmptyTables(t *testing.T) {
	q := string(rune(34))
	schema := "schema = " + q + "https://umono.dev/schemas/theme-attributes/v1" + q + "\n"
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"address missing", schema, "attributes.toml: address: missing"},
		{"address empty", schema + "address = {}\n", "attributes.toml: address: must have at least one address"},
		{"attributes empty", schema + "\n[address." + q + "post" + q + "]\n", "must have at least one attribute"},
	}
	c := fixtureContract(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileAttributes([]byte(tc.doc), c)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
