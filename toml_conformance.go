package themecore

import "fmt"

// nonTOML10Documents are documents that are not TOML v1.0.0: the syntax that
// TOML v1.1.0 adds, and syntax that no TOML version defines.
var nonTOML10Documents = []string{
	"a = {\nb = 1\n}\n",
	"a = { b = 1, }\n",
	"a = \"" + string(rune(backslash)) + "e\"\n",
	"a = \"" + string(rune(backslash)) + "x41\"\n",
	"a = 07:32\n",
	"a = 1979-05-27T07:32Z\n",
	"a = 1\na = 2\n",
	"a = 01\n",
	"a = .5\n",
	"a = 1__2\n",
	"a = [1 2]\n",
	"a b = 1\n",
	"a = 'x\n",
	"a = tru\n",
}

// tomlConformanceDocument is a TOML v1.0.0 document with every kind of value.
const tomlConformanceDocument = `s = "x"
l = 'y'
m = """
z"""
i = -1
f = 1.5
b = true
t = 07:32:00
d = 1979-05-27
dt = 1979-05-27T07:32:00Z
a = [1, 2,]
it = { k = [
1,
] }

[table]
k = 1
`

// TOMLConformance returns an error if the TOML decoder of this binary
// accepts a document that is not TOML v1.0.0 or rejects one that is. The
// go-toml version in a binary is chosen by the whole build, so Umono and CLI
// call it in their tests to check the decoder they are built with.
func TOMLConformance() error {
	for _, doc := range nonTOML10Documents {
		var v map[string]any
		if err := decodeTOML([]byte(doc), &v); err == nil {
			return fmt.Errorf("themecore: the TOML decoder accepts a document that is not TOML v1.0.0: %q", doc)
		}
	}
	var v map[string]any
	if err := decodeTOML([]byte(tomlConformanceDocument), &v); err != nil {
		return fmt.Errorf("themecore: the TOML decoder rejects a TOML v1.0.0 document: %w", err)
	}
	return nil
}
