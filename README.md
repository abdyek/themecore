# themecore

`themecore` is the shared Go package of the [Umono](https://github.com/umono-cms/umono) theme system. The Umono runtime and the Umono CLI embed it, so a theme is checked by the same rules when it is built, installed and activated.

> **Status:** under development for Umono v0.9. The API is not stable yet.

## Install

```sh
go get github.com/umono-cms/themecore
```

## File Format

All configuration files of the theme system (`theme.toml`, `built.toml`, `contract.toml` and `attributes.toml`) are TOML v1.0.0 and UTF-8. Syntax added by newer TOML versions is rejected, and so are keys that a schema does not define.

Each file has a `schema` key. It is required and must be a schema URL that `themecore` reads for that file, for example `https://umono.dev/schemas/theme-manifest/v1` for `theme.toml`. An unknown URL is rejected, not ignored.

Text fields are single line, without control characters and without leading or trailing whitespace.

## Packages

A theme package is read either from a built file (`.umono-theme`, a zip) or from its installed directory:

```go
p, err := themecore.ReadZip(f, size) // size from the file system's metadata
p, err := themecore.ReadDir("themes/acme/minimal/0.1.0")
```

Both check the package limits before any file is read, counting the real bytes while reading, and then the package content allowlist.

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

## License

MIT
