# themecore

`themecore` is the shared Go package of the [Umono](https://github.com/umono-cms/umono) theme system. The Umono runtime and the Umono CLI embed it, so a theme is checked by the same rules when it is built, installed and activated.

> **Status:** under development for Umono v0.9. The API is not stable yet.

## Install

```sh
go get github.com/umono-cms/themecore
```

## File Format

All configuration files of the theme system (`theme.toml`, `built.toml`, `contract.toml` and `attributes.toml`) are TOML v1.0.0 and UTF-8. Syntax added by newer TOML versions is rejected, and so are keys that a schema does not define.

The syntax that TOML v1.1.0 adds is rejected by a scan before decoding, whatever version of the TOML decoder the binary is built with: a newline or a trailing comma in an inline table, the escapes `\e` and `\xHH`, and a time or date-time without seconds. A program that builds `themecore` in calls `TOMLConformance` in its tests:

```go
func TestTOMLConformance(t *testing.T) {
	if err := themecore.TOMLConformance(); err != nil {
		t.Fatal(err)
	}
}
```

It returns an error if the decoder accepts a document that is not TOML v1.0.0 or rejects one that is. Go's version selection cannot cap the version of a dependency, so the decoder in a binary may be newer than the one `themecore` is tested with.

Each file has a `schema` key. It is required and must be a schema URL that `themecore` reads for that file, for example `https://umono.dev/schemas/theme-manifest/v1` for `theme.toml`. An unknown URL is rejected, not ignored.

Text fields are single line, without control characters and without leading or trailing whitespace.

## Packages

A theme package is read either from a built file (`.umono-theme`, a zip) or from its installed directory:

```go
p, err := themecore.ReadZip(f, size) // size from the file system's metadata
p, err := themecore.ReadDir("themes/acme/minimal/0.1.0")
```

Both check the package limits before any file is read, counting the real bytes while reading, and then the package content allowlist.

Hard links are detected by the link count of the file system, which only Unix gives. `ReadDir` therefore fails on other platforms instead of accepting a package it cannot check; `ReadZip` works everywhere.

| Measure | Limit |
| --- | --- |
| Built file (zip) | 64 MB |
| All files unpacked | 64 MB |
| A file under `assets/` | 16 MB |
| `assets/theme.css` | 2 MB |
| `built.toml`, `checksums.json`, `contract.toml`, `attributes.toml` | 1 MB each |
| `LICENSE` | 256 KB |
| Files | 1,000 |
| Directories in a path | 8, `assets` included |
| Path length | 256 bytes |

The package root holds only `built.toml`, `contract.toml`, `checksums.json`, `attributes.toml` and `LICENSE`, and `assets/` is the only directory. Every path segment matches `[a-z0-9][a-z0-9._-]*` and is at most 64 bytes; `LICENSE` is the only exception. Empty directories, symlinks, hard links, duplicate or encrypted zip entries and compression methods other than store and deflate are rejected.

`assets/theme.css` is required and is the only CSS file. Every other file under `assets/` has one of these extensions, and its content must start with the format's magic bytes:

| Extension | Magic bytes |
| --- | --- |
| `.woff2` | `wOF2` |
| `.woff` | `wOFF` |
| `.png` | PNG signature |
| `.jpeg`, `.jpg` | `FF D8 FF` |
| `.webp` | `RIFF`, then `WEBP` at byte 8 |
| `.avif` | An ISO BMFF `ftyp` box with the brand `avif` or `avis` |
| `.gif` | `GIF87a` or `GIF89a` |

## `checksums.json`

`Checksums` returns the content of `checksums.json` during build; `VerifyChecksums` checks a package against it during install and activation:

```go
b, err := themecore.Checksums(p) // p without checksums.json
err = themecore.VerifyChecksums(p)
```

The algorithm is `sha256`, and `files` lists every package file except `checksums.json` itself, with no other entries. `Checksums` is deterministic: keys in byte order and a fixed format. `VerifyChecksums` looks at values, not format.

## `theme.toml`

`ParseManifest` parses and validates `theme.toml`, the manifest the theme author writes:

```go
m, err := themecore.ParseManifest(b)
```

| Key | Required | Rule |
| --- | --- | --- |
| `schema` | Yes | `https://umono.dev/schemas/theme-manifest/v1` |
| `id` | Yes | `org/id`, both parts kebab-case. Not `dev/dev` |
| `contract` | Yes | `org/id@vMAJOR.MINOR`, e.g. `umono/blog@v1.0` |
| `name` | Yes | Not empty, at most 64 characters |
| `version` | Yes | `MAJOR.MINOR.PATCH`, without `v`, pre-release or build metadata |
| `description` | No | At most 280 characters |
| `license` | No | An SPDX license expression. Only the syntax is checked |
| `homepage`, `repository` | No | Absolute `https://` URL, at most 2048 bytes |
| `authors` | No | At least one `[[authors]]` with `name` and an optional `https://` `url` |

Optional keys are `nil` in `Manifest` when they are absent.

## `built.toml`

`ParseBuilt` parses and validates `built.toml`, the manifest of a package. It has the keys of `theme.toml`, with the same rules, under the schema `https://umono.dev/schemas/theme-built/v1`, and two more:

| Key | Required | Rule |
| --- | --- | --- |
| `theme-api-version` | Yes | An integer, 1 or greater |
| `built-at` | Yes | An offset date-time in UTC, e.g. `2026-06-02T00:00:00Z` |

`CheckInstallDir` checks that the install path `themes/{org}/{id}/{version}` matches `id` and `version`:

```go
b, err := themecore.ParseBuilt(data)
err = b.CheckInstallDir("themes/acme/minimal/0.1.0")
```

## `contract.toml`

`ParseContract` parses and validates `contract.toml` under the schema `https://umono.dev/schemas/theme-contract/v1`: first the structural layer below, then the sources, which are parsed with Compono (`compono.Analyze`).

| Key | Required | Rule |
| --- | --- | --- |
| `id` | Yes | `org/id`, both parts kebab-case |
| `version` | Yes | `vMAJOR.MINOR`, e.g. `v1.0` |
| `description` | No | At most 280 characters |
| `layouts` | Yes | At least one `[[layouts]]` with a unique kebab-case `id`, an optional `description` and a non-empty `source` |
| `components` | No | `[[components]]` with a unique `SCREAMING_SNAKE_CASE` `name`, a non-empty `source` and an optional boolean `expose` |

Sources are the only multi-line text fields.

A source with a grammar error that does not depend on values (`invalid-expression`, `unclosed-expression` or `invalid-parameter-definition`), also in a local that is never called, cannot be parsed and is rejected. A published contract never has one, so the error means that the program is older than the contract. It wraps `ErrOutdated`, and CLI and Umono report that they are out of date:

```go
if errors.Is(err, themecore.ErrOutdated) {
	// "CLI is out of date" or "Umono is out of date"
}
```

## Theme CSS

`Transform` turns the theme author's `assets/theme.css` into the canonical CSS that goes into the package; `Verify` checks the canonical CSS of a package:

```go
out, err := themecore.Transform(src) // build
err = themecore.Verify(css)          // install and activation
```

Every style rule of the canonical CSS is inside an `@scope (umono-root)` block, so the theme cannot style anything outside `<umono-root>`. `Transform` parses strictly and never relies on the browser's error recovery; its output always passes `Verify`. `Verify` checks the grammar, not the bytes.

The author grammar is closed:

- **At-rules:** style rules, `@media`, `@supports` and `@layer` blocks; `@layer` statements, `@font-face` and `@keyframes` at the top level only. Every other at-rule, `@import` and `@scope` included, is rejected. No nested CSS.
- **Selectors:** a closed list of pseudo-classes and pseudo-elements. `html`, `body`, `umono-root`, `:root`, `&`, `id` (`#id`, `[id]`, `:target`) and namespace prefixes are rejected. `<umono-root>` itself is styled with `:scope` in the first compound selector, e.g. `:scope.layout-post h1`.
- **`@media`:** the media types `all`, `screen`, `print` and a closed list of media features.
- **Values:** a closed list of value functions (`env()`, `if()`, `sibling-index()` and others are rejected); `attr()` cannot read `id`; `behavior` and `-moz-binding` are rejected.
- **Reserved names:** `umono-` and `compono-` (and `--umono-`, `--compono-`) are reserved for Umono.
- **URLs** are relative paths to allowed files under `assets/`, e.g. `url(fonts/inter.woff2)`.
- **Generated text:** `content`, `quotes`, `list-style-type` and `list-style` cannot generate words. `content` with a string or an image needs alternative text: `content: "→" / "";`.

`FallbackSource` returns the fallback theme, the default look of pages without an active theme, in the author grammar; `FallbackCSS` returns its canonical CSS. `FallbackSource` is also the starting `assets/theme.css` of a new theme.

`Lint` returns warnings for CSS that is valid but never matches, for development tools only. It reports root classes used without `:scope`: a class given only to layout addresses in `attributes.toml` is only on `<umono-root>`, so `.layout-post h1` never matches; write `:scope.layout-post h1`.

## Theme API

The capability ledger (`Features` and `Breaks`) is the single source of the Theme API versions. Rows are only appended.

```go
themecore.ThemeAPICurrent() // newest Theme API version
themecore.ThemeAPIMin()     // oldest Theme API version still accepted
```

`DetectThemeAPIVersion` returns the version a package declares in `built.toml` during build, with the evidence for it. `DetectFeatureFloor` returns the largest version of the features a package uses; install and activation reject a package whose declared version is lower.

```go
version, evidence, err := themecore.DetectThemeAPIVersion(p)
floor, evidence, err := themecore.DetectFeatureFloor(p)
```

Each `Signal` names a feature, the version that introduced it and the first place in the package that shows it.

## License

MIT
