# Kumbuka CLI

`kumbuka-cli` contains the offline and project-oriented tooling for [Kumbuka](https://github.com/kumbuka-me/kumbuka).

The Kumbuka server and CLI are intentionally separate binaries:

- `kumbuka` runs the PostgreSQL-backed web application.
- `kumbuka-cli` builds static sites, creates Git-friendly mirrors, and manages static-site plugin dependencies.

## Commands

```sh
kumbuka-cli build --help
kumbuka-cli serve --help
kumbuka-cli mirror --help
kumbuka-cli plugins --help
```

Build a static site:

```sh
kumbuka-cli build --config kumbuka-site.toml
```

Plugins can publish passive SVGs during the build through the generic SDK artifact API. The CLI writes content-addressed images under `assets/rendered/`; only plugin content that explicitly requests browser interactivity needs a browser renderer at viewing time. No Node.js or Chromium installation is required by CLI users.

Preview a local site with automatic rebuilds and browser reloads:

```sh
kumbuka-cli serve --config kumbuka-site.toml
```

By default, `serve` listens on `127.0.0.1:8000`, builds the site before serving, and watches Markdown, configuration, assets, and the plugin dependency file for changes.
Successful rebuilds reload connected browsers. A failed rebuild leaves the last successfully generated site available.
The preview retains the site's configured URL path (such as `/docs/`) and uses the local server as its origin, including for browser plugins. Restart the server after changing the configured site URL path.

```sh
kumbuka-cli serve --listen-address 127.0.0.1:9000
kumbuka-cli serve --no-build --no-watch
```

`--no-build` serves existing generated files; it does not rewrite URLs or plugin frame policies, so use the default build-enabled mode for an accurate preview. The preview server is intended for local development, not production hosting.
Manage the project's `.kumbukaplugins` file:

```sh
kumbuka-cli plugins list
kumbuka-cli plugins sync
kumbuka-cli plugins add \
  --id com.example.chart \
  --repository example/kumbuka-chart \
  --plugin-version 2.3.0
kumbuka-cli plugins remove --id com.example.chart
```

Create a Markdown mirror of a Kumbuka database:

```sh
kumbuka-cli mirror \
  --database-url 'postgres://USER:PASS@HOST:5432/kumbuka?sslmode=disable' \
  --output kumbuka-mirror
```

## Custom stylesheets

Static navigation defaults to regular links (400), medium sections and actions (500), and semibold active links (600). To override these or other styles, add an optional stylesheet in `kumbuka-site.toml`:

```toml
assets_dir = "assets"
custom_css = "assets/custom.css"
```

Paths are relative to the configuration file. The stylesheet is copied into the site and loaded after built-in and plugin styles on every page, including search and 404 pages. `serve` watches it for changes. A missing file fails the build.

For example, `assets/custom.css` can contain:

```css
:root {
  --nav-link-weight: 500;
  --nav-section-weight: 600;
  --nav-action-weight: 600;
  --nav-active-weight: 700;
}
```

Links include page links in sidebar and top navigation. Sections include folder labels and the documentation label. Actions include navigation controls and header external links. Active links retain their background/color indicators.
Custom CSS can also override colors, typography, spacing, and other selectors; these four navigation variables are the documented weight controls.

Files inside `assets_dir` keep their relative paths under `assets/`; files inside `source_dir` keep their source-relative paths. Other stylesheet files are copied to `assets/<filename>`. Keep fonts and images alongside the stylesheet inside `assets_dir` (or `source_dir`) so relative CSS `url(...)` references are copied too.
Omit `custom_css` to use the defaults.

## Development

The CLI consumes reusable runtime packages from `github.com/kumbuka-me/kumbuka/pkg/...` while keeping all CLI-specific implementations under `internal/`.

For local development of an unreleased server change, use a workspace beside both repositories rather than committing a `replace` directive:

```sh
go work init ./kumbuka ./cli
```

Then build and test normally:

```sh
make build
make test
```

## License

Licensed under the [Apache License 2.0](./LICENSE).
