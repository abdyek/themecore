package themecore

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/umono-cms/compono"
)

// schemaAttributesV1 is the schema URL of attributes.toml.
const schemaAttributesV1 = "https://umono.dev/schemas/theme-attributes/v1"

// layoutScopePath is the scope path of the frames of contract components:
// Umono gives them as sub components of the global UMONO_LAYOUT.
const layoutScopePath = "UMONO_LAYOUT"

// AttributeTable is attributes.toml compiled against a contract: the
// attributes of every address. It is a plain value; rendering reads no file.
type AttributeTable struct {
	addresses map[string]map[string]string
	exposed   map[string]bool
}

// CompileAttributes validates attributes.toml against the contract c and
// compiles it into an AttributeTable. Every address is layout, layout
// BUILTIN or layout COMPONENT BUILTIN with parts separated by a single space,
// and its names are layouts and exposed components of c and built-ins of
// Builtins. A name c defines as a component is never a built-in. Only class
// and sizes are allowed, sizes only on an address that ends with IMAGE.
func CompileAttributes(b []byte, c Contract) (AttributeTable, error) {
	const file = "attributes.toml"
	doc, err := readSchema(file, b, schemaAttributesV1)
	if err != nil {
		return AttributeTable{}, err
	}
	addresses, err := decodeAddresses(b)
	if err != nil {
		return AttributeTable{}, fmt.Errorf("%s: %w", file, err)
	}
	if _, ok := doc["address"]; !ok {
		return AttributeTable{}, fmt.Errorf("%s: address: missing", file)
	}
	if len(addresses) == 0 {
		return AttributeTable{}, fmt.Errorf("%s: address: must have at least one address", file)
	}

	names := newAddressNames(c)
	for _, address := range slices.Sorted(maps.Keys(addresses)) {
		parts, err := names.check(address)
		if err != nil {
			return AttributeTable{}, fmt.Errorf("%s: address.%q: %v", file, address, err)
		}
		attrs := addresses[address]
		if len(attrs) == 0 {
			return AttributeTable{}, fmt.Errorf("%s: address.%q: must have at least one attribute", file, address)
		}
		for _, name := range slices.Sorted(maps.Keys(attrs)) {
			var err error
			switch name {
			case "class":
				err = checkClass(attrs[name])
			case "sizes":
				if len(parts) < 2 || parts[len(parts)-1] != "IMAGE" {
					err = errors.New("only for an address that ends with IMAGE")
				} else {
					err = checkSizes(attrs[name])
				}
			default:
				err = errors.New("not an allowed attribute")
			}
			if err != nil {
				return AttributeTable{}, fmt.Errorf("%s: address.%q: %s: %v", file, address, name, err)
			}
		}
	}
	return AttributeTable{addresses: addresses, exposed: names.exposed}, nil
}

// addressNames are the names the addresses of a contract can hold.
type addressNames struct {
	layouts  map[string]bool
	exposed  map[string]bool
	defined  map[string]bool
	builtins map[string]bool
}

func newAddressNames(c Contract) addressNames {
	n := addressNames{
		layouts:  map[string]bool{},
		exposed:  map[string]bool{},
		defined:  contractDefinedNames(c, analyzeContract(c)),
		builtins: map[string]bool{},
	}
	for _, l := range c.Layouts {
		n.layouts[l.ID] = true
	}
	for _, comp := range c.Components {
		if comp.Expose {
			n.exposed[comp.Name] = true
		}
	}
	for _, name := range Builtins() {
		n.builtins[name] = true
	}
	return n
}

// check returns the parts of address, or an error if it is not one of the
// address forms or holds a name the contract does not give it.
func (n addressNames) check(address string) ([]string, error) {
	parts := strings.Split(address, " ")
	for _, p := range parts {
		if p == "" {
			return nil, errors.New("parts must be separated by a single space")
		}
	}
	if len(parts) > 3 {
		return nil, errors.New("must be layout, layout BUILTIN or layout COMPONENT BUILTIN")
	}
	if !n.layouts[parts[0]] {
		return nil, fmt.Errorf("%s is not a layout of the contract", parts[0])
	}
	if len(parts) == 3 && !n.exposed[parts[1]] {
		return nil, fmt.Errorf("%s is not an exposed component of the contract", parts[1])
	}
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if n.exposed[last] {
			return nil, fmt.Errorf("%s is an exposed component, which has no injection point", last)
		}
		if n.defined[last] || !n.builtins[last] {
			return nil, fmt.Errorf("%s is not a built-in", last)
		}
	}
	return parts, nil
}

// Root returns the attributes of the layout address, which the runtime writes
// on <umono-root>, or nil. It returns a new map on every call.
func (t AttributeTable) Root(layout string) map[string]string {
	return copyAttributes(t.addresses[layout])
}

// Lookup returns the attributes of the built-in rendered with the frame chain
// chain on a page bound to layout, or nil. The scope is the innermost global
// frame whose scope path is [UMONO_LAYOUT] and whose name is an exposed
// component, otherwise the layout; the address is layout [SCOPE] BUILTIN. It
// fits compono.AttributeHookFunc once layout is bound, and returns a new map
// on every call.
func (t AttributeTable) Lookup(layout, builtin string, chain []compono.Frame) map[string]string {
	address := layout + " " + builtin
	for i := len(chain) - 1; i >= 0; i-- {
		f := chain[i]
		if f.Kind == compono.FrameGlobal && len(f.ScopePath) == 1 && f.ScopePath[0] == layoutScopePath && t.exposed[f.Name] {
			address = layout + " " + f.Name + " " + builtin
			break
		}
	}
	return copyAttributes(t.addresses[address])
}

// copyAttributes returns a copy of attrs, or nil if attrs is empty.
func copyAttributes(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	return maps.Clone(attrs)
}
