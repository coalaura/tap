# tap

A small package manager for Windows and Linux (amd64), using a catalog compiled from `packages/*.yml`.

## Commands

```sh
tap tap                 # update tap itself to the latest stable release
tap get go zig bun      # install or update named packages
tap rm bun              # remove named packages
tap ls                  # list installed catalog packages and versions
tap find                # show the complete catalog
tap find compiler       # search names and descriptions
tap info go bun         # show descriptions, sources, paths and installed versions
tap --version           # tap version dev or the version supplied at build time
tap --help
```

`get`, `rm` and `info` accept multiple package names. Names are case-insensitive and unknown names are rejected before any packages are changed. Running `tap` without arguments shows help. `get` uses the latest stable upstream release and leaves an equal or newer installation alone.

`tap tap` checks the latest stable `coalaura/tap` release and replaces the running executable in place, following symlinks. Equal or newer versions are left alone; a `dev` build installs the latest stable release. The download's SHA-256 digest and reported version are verified before replacement. Write access to the executable's directory is required. On Windows, the running executable is renamed to a `.tap-old` backup, restored if replacement fails and cleaned up by the next `tap tap` invocation if it remains locked.

Installed status is discovered from each package's configured path, including installations made by the original upgrader. `ls` shows only catalog packages installed at those paths; `find` searches the entire catalog. `ls`, `find` and `info` work offline. Installed versions are read by invoking each binary with its configured version arguments. Tap does not inspect arbitrary programs elsewhere on `PATH`.

## Install locations

| Package | Windows | Linux |
| --- | --- | --- |
| Go | `%ProgramFiles%\Go` (official MSI) | `/usr/local/go` |
| Zig | `%USERPROFILE%\.zig` | `/usr/local/zig` |
| Bun | `%USERPROFILE%\.bun\bin\bun.exe` | `~/.bun/bin/bun` |
| Other executables | `%USERPROFILE%\.bin\<binary>.exe` | `/usr/local/bin/<binary>` |

Use administrator privileges for Go's Windows MSI and appropriate permissions for Linux's `/usr/local` locations. Under `sudo`, user-local paths refer to the invoking user's home and retain that user's ownership. Put the relevant binary directories on `PATH` to run the installed tools.

`rm` removes a package's executable or the complete toolchain directory for Go and Zig. Bun's cache, configuration and other user data remain in place. On Windows, removing Go downloads the matching official MSI and uninstalls it through Windows Installer; this requires network access and an MSI-installed Go distribution.

## Define packages

Add a file such as `packages/example.yml`:

```yaml
# yaml-language-server: $schema=./schema.json

description: An example command-line tool
repository: owner/example
```

[`packages/schema.json`](packages/schema.json) provides validation, completion and field descriptions in editors using YAML Language Server, including VS Code with the Red Hat YAML extension. Include the schema comment above in new package files. `go generate` remains the final validation step for filename-derived defaults and catalog-wide rules such as unique package names.

The filename supplies the package name and binary name. Defaults are GitHub releases, a `v` tag prefix, `--version` arguments, both supported platforms, assets named `{binary}-{os}-{arch}{ext}` and the standard executable install location above.

For platform-specific archives:

```yaml
# yaml-language-server: $schema=./schema.json

description: An example command-line tool
repository: owner/example
prefix: v
args: [--version]
platforms:
  linux-amd64:
    asset: example-{version}-linux.tar.gz
    executable: example/bin/example
  windows-amd64:
    asset: example-{version}-windows.zip
    executable: example/bin/example.exe
```

Supported fields:

| Field | Meaning |
| --- | --- |
| `name` | Override the filename-derived package name. |
| `description` | Required description used by `find` and `info`. |
| `source` | `github` (default), `go` or `zig`. Go and Zig use official downloads and dedicated toolchain installers. |
| `repository` | GitHub `owner/repository`; required for the GitHub source. |
| `prefix` | Release tag prefix; defaults to `v`. Use `prefix: ''` for bare version tags. |
| `binary` | Executable name without `.exe`; defaults to the filename stem. |
| `args` | Version arguments; defaults to `[--version]` or `[version]` for Go and Zig. |
| `asset` | Default GitHub asset template, overridden by platform entries. |
| `platforms` | Map of `linux-amd64` and/or `windows-amd64` entries. Omit it to support both. |
| `platforms.<platform>.asset` | Platform-specific GitHub asset template. |
| `platforms.<platform>.executable` | Relative executable path within an archive. Omit it to find a unique matching binary filename. |
| `platforms.<platform>.path` | Destination file; defaults to `{bin}/{binary}{ext}`. Must start with `{bin}/` or `{home}/`. |

Asset and archive-executable templates support `{name}`, `{binary}`, `{version}`, `{tag}`, `{os}`, `{arch}` and `{ext}`. `{version}` excludes the tag prefix; `{tag}` includes it. `{ext}` is `.exe` on Windows and empty on Linux. Destination templates additionally support `{bin}` and `{home}`, but cannot contain version-dependent placeholders. Use forward slashes in YAML paths. For example, Bun uses `path: '{home}/.bun/bin/{binary}{ext}'`.

GitHub packages support raw executables and `.zip`, `.tar.gz` or `.tar.xz` archives containing a single executable to install. Additional runtime files are not installed by this source. Archives are extracted within bounded temporary directories, SHA-256 digests are verified and candidate binaries must report the expected version before replacing existing executables. GitHub release assets must provide a SHA-256 `digest`; Go and Zig use their official checksum metadata. Set `GITHUB_TOKEN` when authenticated GitHub API access is needed.

## Generate and build

```sh
go generate ./...
go build -ldflags="-X main.Version=v1.2.3" .
```

`go generate` runs `internal/generate`, parses YAML with `github.com/goccy/go-yaml`, validates definitions and writes the deterministic `packages_gen.go` catalog. Commit the YAML and regenerated Go file together. A normal build uses that generated catalog; Tap needs neither YAML parsing nor external package files at runtime. Regenerate and rebuild after editing a definition.

The CLI uses `github.com/urfave/cli/v3` and operational logging uses `github.com/coalaura/plain/minimal`.

## Releases

Push a `v*` tag, such as `v1.2.3`, to run `.github/workflows/release.yml`. It regenerates the package catalog, builds CPU-compatible Linux and Windows amd64 binaries and injects the tag into `main.Version`. Configure the `COALAURA_PFX` and `COALAURA_PFX_PASSWORD` repository secrets for signing. The workflow publishes signed `tap-linux-amd64` and `tap-windows-amd64.exe` binaries, `SHA256SUMS`, build provenance and generated release notes. Tags containing a hyphen, such as `v1.2.3-rc.1`, produce prereleases.
