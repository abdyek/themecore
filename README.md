# themecore

`themecore` is the shared Go package of the [Umono](https://github.com/umono-cms/umono) theme system. The Umono runtime and the Umono CLI embed it, so a theme is checked by the same rules when it is built, installed and activated.

> **Status:** under development for Umono v0.9. The API is not stable yet.

## Install

```sh
go get github.com/umono-cms/themecore
```

## File Format

All configuration files of the theme system (`theme.toml`, `built.toml`, `contract.toml` and `attributes.toml`) are TOML v1.0.0 and UTF-8. Syntax added by newer TOML versions is rejected, and so are keys that a schema does not define.

## License

MIT
