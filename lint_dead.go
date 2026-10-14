package themecore

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// deadAddressWarnings returns a warning for every layout COMPONENT BUILTIN
// address of addrs whose component the layout never renders: the component
// is not reachable from the layout's source through the references of the
// contract sources (calls, values and defaults, which may be rendered), or
// nil. Address validation only checks that names exist.
func deadAddressWarnings(p Package, addrs map[string]map[string]string) []Warning {
	b, ok := p.Files["contract.toml"]
	if !ok {
		return nil
	}
	var c Contract
	if err := decodeTOML(b, &c); err != nil {
		return nil
	}
	sources := analyzeContract(c)
	components := map[string]int{}
	for i, comp := range c.Components {
		components[comp.Name] = len(c.Layouts) + i
	}

	var warnings []Warning
	for _, address := range slices.Sorted(maps.Keys(addrs)) {
		parts := strings.Split(address, " ")
		if len(parts) != 3 {
			continue
		}
		layout := -1
		for i, l := range c.Layouts {
			if l.ID == parts[0] {
				layout = i
			}
		}
		if layout < 0 || reachesComponent(sources, components, layout, parts[1]) {
			continue
		}
		warnings = append(warnings, Warning{
			Message: fmt.Sprintf("%s never matches: the layout %s never renders %s", address, parts[0], parts[1]),
			Where:   address,
		})
	}
	return warnings
}

// reachesComponent reports whether the source at index from of sources
// reaches the component name. components maps component names to the index
// of their source.
func reachesComponent(sources []contractSource, components map[string]int, from int, name string) bool {
	seen := map[int]bool{from: true}
	queue := []int{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, ref := range sources[cur].analysis.Refs {
			if ref.Local {
				continue
			}
			next, ok := components[ref.Name]
			if !ok {
				continue
			}
			if ref.Name == name {
				return true
			}
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}
