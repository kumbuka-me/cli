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

For documents using the Mermaid plugin, static builds pre-render ordinary `mermaid`
fences to light/dark SVG assets. Install the matching Mermaid CLI **12.0.0**
(`npm install -g @mermaid-js/mermaid-cli@12.0.0`) and ensure `mmdc` is
on `PATH`, or set `MERMAID_CLI` to its executable path. No Mermaid CLI is
required for sites without ordinary Mermaid diagrams.

Add `{mermaid interactive="true"}` on a line after a Mermaid fence, or put
`%% kumbuka-interactive: true` inside the fence, to retain the isolated
Mermaid.js renderer on the generated site. Kumbuka's normal web application
is unchanged and continues to render diagrams using Mermaid.js.

Generated SVGs use content-addressed filenames and are reused across pages
within a build. Static builds fail with a filename-specific error if Mermaid
CLI is missing, the version does not match, or a diagram cannot be rendered.

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
