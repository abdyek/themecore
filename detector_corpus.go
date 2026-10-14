package themecore

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// detectorCorpus holds the cases of the detector corpus: every directory of
// detectorcorpus is a case with a contract.toml, an optional attributes.toml
// and signals.golden, the signals the case must give, one "feature where"
// per line in ledger order.
//
//go:embed detectorcorpus
var detectorCorpus embed.FS

// DetectorCorpus runs the detectors on every case of the detector corpus and
// returns an error for the first case whose signals differ from its golden
// file or whose contract does not parse. The detectors read contracts with
// Compono, so Umono and CLI call it in their tests: a Compono version that
// changes the signals fails there.
func DetectorCorpus() error {
	entries, err := fs.ReadDir(detectorCorpus, "detectorcorpus")
	if err != nil {
		return fmt.Errorf("themecore: detector corpus: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if err := runDetectorCorpusCase(name); err != nil {
			return fmt.Errorf("themecore: detector corpus %s: %w", name, err)
		}
	}
	return nil
}

// runDetectorCorpusCase runs the detectors on the case name.
func runDetectorCorpusCase(name string) error {
	dir := path.Join("detectorcorpus", name)
	p := Package{Files: map[string][]byte{}}
	for _, file := range []string{"contract.toml", "attributes.toml"} {
		b, err := fs.ReadFile(detectorCorpus, path.Join(dir, file))
		if err != nil {
			continue
		}
		p.Files[file] = b
	}
	if _, err := ParseContract(p.Files["contract.toml"]); err != nil {
		return err
	}
	want, err := fs.ReadFile(detectorCorpus, path.Join(dir, "signals.golden"))
	if err != nil {
		return err
	}
	_, evidence, err := DetectFeatureFloor(p)
	if err != nil {
		return err
	}
	lines := make([]string, 0, len(evidence))
	for _, s := range evidence {
		lines = append(lines, s.Feature+" "+s.Where)
	}
	got := strings.Join(lines, "\n")
	if got != strings.TrimSpace(string(want)) {
		return fmt.Errorf("signals differ:\ngot:\n%s\nwant:\n%s", got, strings.TrimSpace(string(want)))
	}
	return nil
}
