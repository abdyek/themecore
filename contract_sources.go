package themecore

import (
	"errors"
	"fmt"

	"github.com/umono-cms/compono"
)

// ErrOutdated is wrapped by the error for a contract that uses Compono syntax
// or a component that this themecore does not know. A published contract has
// neither, so the program is older than the contract: CLI reports that CLI
// is out of date, Umono that Umono is out of date.
var ErrOutdated = errors.New("themecore is older than the contract")

// contractSource is a source of a contract and its analysis.
type contractSource struct {
	key      string
	analysis compono.Analysis
}

// analyzeContract analyzes the sources of c with Compono: the layouts, then
// the components, in order.
func analyzeContract(c Contract) []contractSource {
	sources := make([]contractSource, 0, len(c.Layouts)+len(c.Components))
	for i, l := range c.Layouts {
		analysis, _ := compono.Analyze([]byte(l.Source))
		sources = append(sources, contractSource{key: fmt.Sprintf("layouts[%d].source", i), analysis: analysis})
	}
	for i, comp := range c.Components {
		analysis, _ := compono.Analyze([]byte(comp.Source))
		sources = append(sources, contractSource{key: fmt.Sprintf("components[%d].source", i), analysis: analysis})
	}
	return sources
}

// syntaxErrorCodes are the codes of the value-independent grammar errors.
// Compono never fails to parse: a source with one of them is a source it
// cannot parse.
var syntaxErrorCodes = map[compono.DiagnosticCode]bool{
	compono.CodeInvalidExpression:          true,
	compono.CodeUnclosedExpression:         true,
	compono.CodeInvalidParameterDefinition: true,
}

// checkContractSyntax returns an error for the first grammar error of the
// sources, uncalled locals included.
func checkContractSyntax(sources []contractSource) error {
	for _, s := range sources {
		for _, d := range s.analysis.Diagnostics {
			if syntaxErrorCodes[d.Code] {
				return fmt.Errorf("contract.toml: %s: %d:%d: %s: %w", s.key, d.Range.Start.Line, d.Range.Start.Column, d.Code, ErrOutdated)
			}
		}
	}
	return nil
}
