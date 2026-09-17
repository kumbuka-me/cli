# Kumbuka CLI

`kumbuka-cli` contains the offline and project-oriented tooling for [Kumbuka](https://github.com/kumbuka-me/kumbuka).

The Kumbuka server and CLI are intentionally separate binaries:

- `kumbuka` runs the PostgreSQL-backed web application.
- `kumbuka-cli` builds static sites, creates Git-friendly mirrors, and manages static-site plugin dependencies.

## Commands

```sh
kumbuka-cli build --help
kumbuka-cli mirror --help
kumbuka-cli plugins --help
```

Build a static site:

```sh
kumbuka-cli build --config kumbuka-site.toml
```

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
