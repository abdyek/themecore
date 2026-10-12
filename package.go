package themecore

// Package is the content of a theme package. During build it is the state that
// goes into the built file, canonical CSS included. During install and
// activation it is the package itself, read from the built file or from the
// installed directory.
type Package struct {
	// Files maps the path of each file to its content. Paths are relative to
	// the package root and use "/" as the separator. Directories are not
	// listed; they follow from the paths.
	Files map[string][]byte
}
