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

Plugins may opt into **generic static browser exports** through an optional
`assets/static-exports.json` declaration and `exportStatic` method in their
existing browser module. During a build, exported passive SVGs are validated
and published as content-addressed light/dark assets. Plugins can return
`null` to retain their regular interactive browser rendering.

Static export requires Node.js, a local Playwright installation, and Chromium
in the site-build project when an exportable block is present:

```sh
npm install --save-dev playwright@1.65.0
npx playwright install chromium
```

The CLI does not depend on Mermaid CLI or any plugin-specific command. The
plugin's own browser script handles which content is eligible and how it renders.
Sites without exportable plugin blocks do not start a headless browser.

Preview a local site with automatic rebuilds and browser reloads:

```sh
kumbuka-cli serve --config kumbuka-site.toml
```

By default, `serve` listens on `127.0.0.1:8000`, builds the site before serving,
and watches Markdown, configuration, assets, and the plugin dependency file for
changes. Successful rebuilds reload connected browsers. A failed rebuild leaves
the last successfully generated site available. The preview retains the site's
configured URL path (such as `/docs/`) and uses the local server as its origin,
including for browser plugins. Restart the server after changing the configured
site URL path.

```sh
kumbuka-cli serve --listen-address 127.0.0.1:9000
kumbuka-cli serve --no-build --no-watch
```

`--no-build` serves existing generated files; it does not rewrite URLs or plugin
frame policies, so use the default build-enabled mode for an accurate preview.
The preview server is intended for local development, not production hosting.

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
