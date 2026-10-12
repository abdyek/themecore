package themecore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// checksumsJSON is the path of the integrity record of a package.
const checksumsJSON = "checksums.json"

// checksumsFile is the content of checksums.json.
type checksumsFile struct {
	Algorithm string            `json:"algorithm"`
	Files     map[string]string `json:"files"`
}

// Checksums returns the content of checksums.json for p: the sha256 of every
// file except checksums.json itself. The output is deterministic: the same
// files give the same bytes, and so the same release.
func Checksums(p Package) ([]byte, error) {
	hashes := make(map[string]string, len(p.Files))
	paths := make([]string, 0, len(p.Files))
	for path := range p.Files {
		if path == checksumsJSON {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		sum := sha256.Sum256(p.Files[path])
		hashes[path] = hex.EncodeToString(sum[:])
	}
	v := checksumsFile{Algorithm: "sha256", Files: hashes}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// VerifyChecksums checks p against its checksums.json. It looks at the
// values, not the format: a reformatted file with the right values is
// accepted.
func VerifyChecksums(p Package) error {
	content, ok := p.Files[checksumsJSON]
	if !ok {
		return fmt.Errorf("package: %s: missing", checksumsJSON)
	}
	dec := json.NewDecoder(bytes.NewReader(content))
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("package: %s: not valid JSON", checksumsJSON)
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("package: %s: not valid JSON", checksumsJSON)
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k != "algorithm" && k != "files" {
			return fmt.Errorf("package: %s: unknown key %q", checksumsJSON, k)
		}
	}
	var cf checksumsFile
	if err := json.Unmarshal(content, &cf); err != nil {
		return fmt.Errorf("package: %s: not valid JSON", checksumsJSON)
	}
	if cf.Algorithm != "sha256" {
		return fmt.Errorf("package: %s: algorithm must be sha256", checksumsJSON)
	}
	listed := make([]string, 0, len(cf.Files))
	for path := range cf.Files {
		listed = append(listed, path)
	}
	sort.Strings(listed)
	for _, path := range listed {
		if path == checksumsJSON {
			return fmt.Errorf("package: %s: lists itself", checksumsJSON)
		}
		file, ok := p.Files[path]
		if !ok {
			return fmt.Errorf("package: %s: %s: not in the package", checksumsJSON, path)
		}
		sum := sha256.Sum256(file)
		got := hex.EncodeToString(sum[:])
		if got != cf.Files[path] {
			return fmt.Errorf("package: %s: checksum does not match", path)
		}
	}
	all := make([]string, 0, len(p.Files))
	for path := range p.Files {
		if path == checksumsJSON {
			continue
		}
		all = append(all, path)
	}
	sort.Strings(all)
	for _, path := range all {
		if _, ok := cf.Files[path]; !ok {
			return fmt.Errorf("package: %s: missing from checksums.json", path)
		}
	}
	return nil
}
