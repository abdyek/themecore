# themecore

[![coverage](https://raw.githubusercontent.com/umono-cms/themecore/badges/coverage.svg)](https://github.com/umono-cms/themecore/actions/workflows/test.yml?query=branch%3Amain)

`themecore` is the shared Go package of the [Umono](https://github.com/umono-cms/umono) theme system. The Umono runtime and the Umono CLI embed it, so a theme is checked by the same rules when it is built, installed and activated.

> **Status:** under development for Umono v0.9. The API is not stable yet.

This README describes the rules a theme is held to and when they apply. The functions and their signatures are documented in the package documentation.

## Install

```sh
go get github.com/umono-cms/themecore
```

## Where the Checks Run

The checks of a theme run from one place, in a fixed order, through three entry points. CLI and Umono do not run single checks, so no check runs on its own, past the package limits, the content allowlist and `schema`.

- **Build and the dev proxy** (`CheckSource`) check the development directory against the contract that `theme.toml` refers to. Only `assets/`, `attributes.toml` and `LICENSE` are read into the package, with the contract as `contract.toml`. `theme.toml` is checked but does not go in, and other files such as `.dev/` and `dist/` are not read. An empty directory is skipped; any other file that breaks the package rules stops the build instead of being left out silently. `assets/theme.css` is replaced by its canonical CSS. The result is what goes into the built file: the CLI adds `built.toml` with the declared Theme API version and then `checksums.json`. Build also gives the evidence of the version and [warnings](#warnings) that do not stop it.
- **Install and activation** (`CheckPackage`) check a package read from a built file or an installed directory. `checksums.json` and the Theme API range are checked by the caller before. Then the required files must exist (`built.toml`, `contract.toml`, `checksums.json`, `assets/theme.css`), `built.toml` and `contract.toml` must be valid and name the same contract, and the assets, the canonical CSS, the files its URLs point to, `attributes.toml` and `LICENSE` must pass. Last, a `theme-api-version` lower than the version of the features the package uses is rejected. The result is the contract, the compiled attribute table (empty without `attributes.toml`) and the feature floor with its evidence.
- **Umono's dev mode** (`CheckDev`) checks `themes/dev/dev/0.0.0/` on every request and keeps no cache. It runs the checks of install and activation with the privileges of `dev/dev`, which exist nowhere else:
  - There is no `checksums.json`.
  - `assets/`, `attributes.toml` and `contract.toml` may be symlinks, but they must resolve inside the theme project.
  - `assets/theme.css` is turned into canonical CSS as in build, instead of being checked as canonical CSS.
  - `built.toml` has the id `dev/dev` and the version `0.0.0`, which every other path rejects.

  The package limits and content, `built.toml`, `contract.toml`, `LICENSE` and the feature floor are fatal: the theme is not activated. Errors of `theme.css`, `attributes.toml` and single assets are returned one by one, so dev mode goes on: the `theme.css` request fails, pages render without attribute injection, and a broken asset is not served. The result also holds the validated `built.toml` and the unchanged bytes of `contract.toml`, so Umono never reads those files itself.

Checks that need a database or the network, such as downloading the [signature of the contract](#contract-signatures), belong to the Umono runtime and are not in `themecore`.

## File Format

All configuration files of the theme system (`theme.toml`, `built.toml`, `contract.toml` and `attributes.toml`) are TOML v1.0.0 and UTF-8. Syntax added by newer TOML versions is rejected, and so are keys that a schema does not define. The syntax that TOML v1.1.0 adds is rejected by a scan before decoding, whatever version of the TOML decoder the binary is built with: a newline or a trailing comma in an inline table, the escapes `\e` and `\xHH`, and a time or date-time without seconds.

Each file has a `schema` key. It is required and must be a schema URL that `themecore` reads for that file, for example `https://umono.dev/schemas/theme-manifest/v1` for `theme.toml`. An unknown URL is rejected, not ignored.

Text fields are single line, without control characters and without leading or trailing whitespace. Contract sources are the only multi-line text fields.

## Packages

A theme package is a built file (`.umono-theme`, a zip) or its installed directory, `themes/{org}/{id}/{version}/`. Both are read the same way: the package limits are checked before any file is parsed, by counting the real bytes while reading, and then the content allowlist. A built file larger than the limit is not opened.

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

The limits are fixed and the same in build, install, activation and dev mode.

The package root holds only `built.toml`, `contract.toml`, `checksums.json`, `attributes.toml` and `LICENSE`, and `assets/` is the only directory. Every path segment matches `[a-z0-9][a-z0-9._-]*` and is at most 64 bytes; `LICENSE` is the only exception. Empty directories, symlinks, hard links, duplicate or encrypted zip entries and compression methods other than store and deflate are rejected. An installed directory is checked again on activation, so a file dropped into it later makes the theme invalid. `LICENSE` must be valid UTF-8 and is not checked otherwise.

Hard links are detected by the link count of the file system, which only Unix gives. Reading an installed directory therefore fails on other platforms instead of accepting a package it cannot check; reading a built file works everywhere.

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

Build writes `checksums.json`; install and activation verify the package against it. The algorithm is `sha256`, and `files` lists every package file except `checksums.json` itself, with no other entries. The hash of `assets/theme.css` is that of the canonical CSS. Writing is deterministic, with keys in byte order and a fixed format, so the same content gives the same bytes. Verifying looks at values, not format.

## `theme.toml`

`theme.toml` is the manifest the theme author writes. It does not go into the package.

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

## `built.toml`

`built.toml` is the manifest of a package, written by the CLI. It has the keys of `theme.toml`, with the same rules, under the schema `https://umono.dev/schemas/theme-built/v1`, and two more:

| Key | Required | Rule |
| --- | --- | --- |
| `theme-api-version` | Yes | An integer, 1 or greater |
| `built-at` | Yes | An offset date-time in UTC, e.g. `2026-06-02T00:00:00Z` |

An installed package sits at `themes/{org}/{id}/{version}`, which must match its `id` and `version`.

## `contract.toml`

`contract.toml` is validated under the schema `https://umono.dev/schemas/theme-contract/v1`: first the structural layer below, then the sources, which are parsed with Compono.

| Key | Required | Rule |
| --- | --- | --- |
| `id` | Yes | `org/id`, both parts kebab-case |
| `version` | Yes | `vMAJOR.MINOR`, e.g. `v1.0` |
| `description` | No | At most 280 characters |
| `layouts` | Yes | At least one `[[layouts]]` with a unique kebab-case `id`, an optional `description` and a non-empty `source` |
| `components` | No | `[[components]]` with a unique `SCREAMING_SNAKE_CASE` `name`, a non-empty `source` and an optional boolean `expose` |

A source with a grammar error that does not depend on values (`invalid-expression`, `unclosed-expression` or `invalid-parameter-definition`), also in a local that is never called, cannot be parsed and is rejected. So is a call or a value that resolves to nothing: not to a local of its source, a component of the contract or a built-in open to themes. A default (`content = NO_MATTER`) only names the type of a parameter and is not checked. A published contract has neither, so the error means that the program is older than the contract. The error wraps `ErrOutdated`, so CLI and Umono can report that they are out of date instead of showing the raw error.

## Contract Signatures

Every published contract is signed with [minisign](https://jedisct1.github.io/minisign/). The signature is a separate file next to the contract in the registry, at the URL of the contract with `.minisig` appended, for example `https://umono.dev/registry/v1/contracts/umono/page/v0.1.minisig`. It is not part of the package: the caller downloads it, reads at most 4 KiB of the response, and gives `VerifyContractSignature` the bytes of the package's `contract.toml`, the bytes of the signature and the reference the package declares. The function does not touch the network, the file system or the clock.

The public key is embedded in `themecore` and cannot be replaced by the caller. It is the second line of `umono.pub`:

```
untrusted comment: minisign public key FF93F835DAA967E3
RWTjZ6naNfiT/6JEdZFEcmm1K2n/xSsuLy9zcIaZ37Rxo2XVh1R3vs1d
```

A signature is accepted only when all of these hold:

1. The file has four lines separated by `\n`; a final newline is optional. Line 1 starts with `untrusted comment: `, and the rest of it is ignored. Line 3 starts with `trusted comment: `.
2. Line 2 is base64 of 74 bytes: the algorithm (2), the key id (8) and the signature (64).
3. The algorithm is `ED`, the signature of the BLAKE2b-512 hash of the file. This is the default of minisign 0.10 and later. `Ed`, the legacy format that signs the file itself, is rejected.
4. The key id is that of the embedded key.
5. The signature verifies with Ed25519 over the BLAKE2b-512 hash of `contract.toml`.
6. Line 4 is base64 of 64 bytes, the global signature. It verifies over the signature of line 2 followed by the text of the trusted comment.
7. The trusted comment is exactly `contract ` followed by the reference, for example `contract umono/page@v0.1`, so a signed contract cannot be served under the path of another contract.

Every error wraps `ErrContractSignature`. The text names the step that failed, for tests and logs. The site owner gets one answer for all of them: the contract in the package is not the published contract. Failing to download the signature is the caller's error and never reaches this function.

## `attributes.toml`

`attributes.toml` is validated under the schema `https://umono.dev/schemas/theme-attributes/v1` against the package's contract and compiled into a table that rendering reads without touching files.

- An address is `layout`, `layout BUILTIN` or `layout COMPONENT BUILTIN`, with parts separated by a single space. The layout is a layout of the contract, the component is an exposed one and the built-in is one open to themes. A name the contract defines as a component is never a built-in. `layout COMPONENT` is rejected: an exposed component has no element of its own to write to.
- Only `class` and `sizes` are allowed. `class` is a list of unique printable ASCII tokens separated by single spaces, without the `umono-` and `compono-` prefixes (case-insensitive), of at most 1024 bytes. `sizes` is allowed only on an address that ends with `IMAGE` and is an HTML source size list without `auto`. Its media conditions use the media feature vocabulary of the theme CSS, its sizes are lengths in its units or `calc()`, `min()`, `max()` and `clamp()` of them, and percentages are rejected.
- The attributes of a `layout` address go on `<umono-root>`, which the runtime writes itself. The other addresses reach Compono through its attribute hook.
- The scope of a rendered built-in comes from its frame chain: the innermost global frame whose scope path is `[UMONO_LAYOUT]` and whose name is an exposed component, otherwise the layout. A local frame, or a site global with the name of an exposed component, is not a scope.

## Theme CSS

Build turns the theme author's `assets/theme.css` into the canonical CSS that goes into the package; install and activation check that the CSS of a package is canonical. Every style rule of the canonical CSS is inside an `@scope (umono-root)` block, so the theme cannot style anything outside `<umono-root>`. The author's CSS is parsed strictly, never relying on the browser's error recovery, and the canonical CSS that build makes always passes the check of install and activation. That check looks at the grammar, not the bytes, so the serialization format can change without rejecting older packages.

The author grammar is closed:

- **At-rules:** style rules, `@media`, `@supports` and `@layer` blocks; `@layer` statements, `@font-face` and `@keyframes` at the top level only. Every other at-rule, `@import` and `@scope` included, is rejected. No nested CSS.
- **Selectors:** a closed list of pseudo-classes and pseudo-elements. `html`, `body`, `umono-root`, `:root`, `&`, `id` (`#id`, `[id]`, `:target`) and namespace prefixes are rejected. `<umono-root>` itself is styled with `:scope` in the first compound selector, e.g. `:scope.layout-post h1`.
- **`@media`:** the media types `all`, `screen`, `print` and a closed list of media features.
- **Values:** a closed list of value functions (`env()`, `if()`, `sibling-index()` and others are rejected); `attr()` cannot read `id`; `behavior` and `-moz-binding` are rejected.
- **Reserved names:** `umono-` and `compono-` (and `--umono-`, `--compono-`) are reserved for Umono.
- **URLs** are relative paths to allowed files under `assets/`, e.g. `url(fonts/inter.woff2)`.
- **Generated text:** `content`, `quotes`, `list-style-type` and `list-style` cannot generate words. `content` with a string or an image needs alternative text: `content: "→" / "";`.

Comments do not go into the canonical CSS, `/*! ... */` license notes included; a license note belongs in `LICENSE`.

`themecore` also carries the fallback theme, the default look of pages without an active theme, in the author grammar and as canonical CSS. Its source is the starting `assets/theme.css` of a new theme.

### Warnings

Build warns about CSS that is valid but never matches. Warnings are for development tools only and do not stop the build; install and activation do not look for them.

- **Root classes without `:scope`.** A class given only to layout addresses in `attributes.toml` is only on `<umono-root>`, so `.layout-post h1` never matches; write `:scope.layout-post h1`.
- **Addresses the layout never renders.** `post-list MAIN_MENU LINK` never matches when `MAIN_MENU` cannot be reached from the source of `post-list` through calls, values and defaults of the contract sources.

## Theme API

The capability ledger inside `themecore` is the single source of the Theme API versions. It has feature rows, each with the version that introduced it and a detector, and break rows for narrowings. Rows are only appended, and the ledger cannot be changed from outside: it shows only through the Theme API versions and the detection results. The newest version is the largest version of all rows; the oldest version still accepted is the largest version of the break rows.

The feature floor of a package is the largest version of the features it uses, 0 if it uses none. Build declares the larger of the floor and the oldest accepted version as `theme-api-version`; install, activation and dev mode compute the floor again and reject a package whose declared version is lower. Both look at the package, the canonical CSS included, not at the author's source. The evidence of the floor names each feature used, the version that introduced it and the first place in the package that shows it.

The built-ins and Markdown elements open to themes are rows too (`compono.builtin.*`, `compono.markdown.*`), and the sets derive from them: `LINK`, `IMAGE`, `NAV` and the other built-ins, `heading`, `p`, `br` and the other Markdown elements. Address validation, the detectors and the undefined call rule look only at these sets, not at Compono's lists, so a built-in Compono adds stays closed to themes until its row is added.

A built-in detector looks for a reference in `contract.toml` that resolves to the built-in (a call, a value or a default) and for an address of `attributes.toml` whose type filter is it. A name the contract defines as a component, in `[[components]]` or as a local, is never a built-in. A Markdown detector looks for the element in `contract.toml`. The place of a signal in `contract.toml` is the key of the source and the line in it, such as `contract.toml:layouts[0].source:4`.

Theme API 2 adds `__auto` for the `current` of `LINK` (`compono.link.current-auto`): the application marks the active page instead of the contract. Only `__auto` written in `contract.toml` raises the floor to 2; a value from page data, a parameter or a record leaves the output the theme sees unchanged and is not detected, so a theme that does not write it still declares 1. Compono does not report the line of a directive, so the place of this signal is the key of the source only, such as `contract.toml:layouts[0].source`.

## Tests of an Embedding Program

Go's version selection cannot cap the version of a dependency, so a program that builds `themecore` in may have a newer TOML decoder or a newer Compono than the ones `themecore` is tested with. Three checks run with the dependencies of the program, and the program runs them in its tests:

```go
func TestThemecore(t *testing.T) {
	checks := []struct {
		name  string
		check func() error
	}{
		{"TOML conformance", themecore.TOMLConformance},
		{"detector corpus", themecore.DetectorCorpus},
		{"built-in set", themecore.BuiltinSet},
	}
	for _, c := range checks {
		if err := c.check(); err != nil {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}
```

- **TOML conformance** fails if the decoder accepts a document that is not TOML v1.0.0 or rejects one that is.
- **The detector corpus** (`detectorcorpus/`: contract inputs and the signals each must give) fails if a contract of the corpus does not parse or gives other signals.
- **The built-in set** fails if Compono no longer has a built-in or Markdown element of the sets: a removal without a break row. Compono's extra names are allowed and stay closed to themes.

## License

MIT
