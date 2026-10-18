# tap

A small package manager for Windows and Linux (amd64), using a catalog compiled from `packages/*.yml`.

## Install

### Linux

```sh
curl -fsSL https://src.ws2.sh/tap/install.sh | sh
```

Installs to `/usr/local/bin/tap`, using `sudo` when not running as root. Requires Linux amd64, `curl` and standard coreutils (including `sha256sum`). If `/usr/local/bin` is missing from `PATH`, the installer prints the command to add it to your shell.

### Windows

```powershell
irm https://src.ws2.sh/tap/install.ps1 | iex
```

Installs to `%USERPROFILE%\.bin\tap.exe` and adds `%USERPROFILE%\.bin` to your user and current PowerShell session `PATH`. Requires Windows amd64 and PowerShell 5.1 or newer; administrator privileges are not needed.

Both scripts download the latest stable [GitHub release](https://github.com/coalaura/tap/releases/latest) and verify the binary against its release's `SHA256SUMS` before installation. Run `tap --help` to get started and `tap tap` to update later (with `sudo tap tap` for the default Linux installation).

## Commands

```sh
tap tap                 # update tap itself to the latest stable release
tap get go zig bun      # install or update named packages
tap get onino           # install or update the default CPU variant
tap get onino:gpu       # install or switch to the GPU variant
tap get onino:cpu       # explicitly select the CPU variant
tap get --force onino   # reinstall without checking the installed version
tap rm bun              # remove named packages
tap ls                  # list installed catalog packages and versions
tap find                # show the complete catalog
tap find compiler       # search names and descriptions
tap info go bun         # show descriptions, sources, paths and installed versions
tap --version           # tap version dev or the version supplied at build time
tap --help
```

`get`, `rm` and `info` accept multiple case-insensitive package names, optionally suffixed with `:variant`. Unknown names and conflicting variants are rejected before changes. Running `tap` without arguments shows help. `get` uses the latest stable upstream release and leaves an equal or newer installation of the selected variant alone.

Use `tap get --force <package[:variant]>...` to reinstall the latest stable release without checking the installed version or variant, including when an older binary does not support its version flag. This can replace a newer installed version. Download verification and validation of the new binaries still apply.

Variants share an install path, so selecting another replaces the installed variant. Tap records installed variants in `~/.config/tap.yml` on both Windows and Linux.

`tap tap` checks the latest stable `coalaura/tap` release and replaces the running executable in place, following symlinks. Equal or newer versions are left alone; a `dev` build installs the latest stable release. The download's SHA-256 digest and reported version are verified before replacement. The version check runs from a staging directory beside the installed executable, so temporary directories can be mounted with `noexec`. Write access to the executable's directory is required. On Windows, the running executable is renamed to a `.tap-old` backup, restored if replacement fails and cleaned up by the next `tap tap` invocation if it remains locked.

Installed status is discovered from each package's configured path, including installations made by the original upgrader. `ls` shows only catalog packages installed at those paths; `find` searches the entire catalog. `ls`, `find` and `info` work offline. Installed versions are read by invoking each binary with its configured version arguments, except PACE, which uses embedded Go build metadata. Tap does not inspect arbitrary programs elsewhere on `PATH`.

## Install locations

| Package | Windows | Linux |
| --- | --- | --- |
| Go | `%ProgramFiles%\Go` (official MSI) | `/usr/local/go` |
| PACE | `%ProgramFiles%\Go\bin` | `/usr/local/go/bin` |
| Zig | `%USERPROFILE%\.zig` | `/usr/local/zig` |
| Bun | `%USERPROFILE%\.bun\bin\bun.exe` | `~/.bun/bin/bun` |
| Rio | `%ProgramFiles%\Rio\rio.exe` (official MSI) | Not supported |
| Other executables | `%USERPROFILE%\.bin\<binary>.exe` | `/usr/local/bin/<binary>` |

Use administrator privileges for Windows MSI installers and appropriate permissions for Linux's `/usr/local` locations. Under `sudo`, user-local paths refer to the invoking user's home and retain that user's ownership. Put the relevant binary directories on `PATH` to run the installed tools.

Tap warns before running commands if `%USERPROFILE%\.bin` on Windows or `/usr/local/bin` on Linux is missing from `PATH`.

`rm` removes a package's executables or the complete toolchain directory for Go and Zig. Bun's cache, configuration and other user data remain in place. On Windows, removing Go or Rio downloads the matching official MSI and uninstalls it through Windows Installer; this requires network access and an MSI-installed distribution.

`findutils` installs `find`, `locate`, `updatedb` and `xargs` together. `diffutils` installs its multi-call binary, used as `diffutils diff ...` or `diffutils cmp ...`. `age` installs both `age` and `age-keygen`. Companion executables are installed beside the primary binary and removed with the package; a missing companion triggers reinstallation on the next `get`.

`pace` installs `pace`, `compilepe` and `asmpe` from the archive's `bin` directory beside Tap's Go binary, with `.exe` suffixes on Windows. PACE requires the exact matching Go release to be installed at Tap's Go location before installation; use `tap get go pace` to update both in order. Removing PACE removes only those three executables. Replacing Go's directory on Linux also removes the overlay, so run `tap get pace` afterwards. PACE's version is read from embedded Go build metadata because its version command requires a matching Go installation, allowing Tap to list and update an overlay after Go changes.

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
| `source` | `github` (default), `msi`, `go`, `pace` or `zig`. MSI uses a GitHub-hosted Windows installer; PACE uses a GitHub archive and Go's bin directory; Go and Zig use official downloads and dedicated toolchain installers. |
| `repository` | GitHub `owner/repository`; required for the GitHub, MSI and PACE sources. |
| `prefix` | Release tag prefix; defaults to `v`. Use `prefix: ''` for bare version tags. |
| `binary` | Executable name without `.exe`; defaults to the filename stem. |
| `companions` | Additional executable names in a verified GitHub or PACE archive, installed beside the primary binary; defaults to none, except PACE requires `compilepe` and `asmpe`. |
| `args` | Version arguments; defaults to `[--version]` or `[version]` for Go, PACE and Zig. PACE uses embedded Go build metadata instead of invoking these arguments. |
| `asset` | Default GitHub asset template, overridden by platform entries. |
| `checksums` | Release asset containing GNU-style SHA-256 sums, used when the download asset has no GitHub digest. |
| `default_variant` | Variant selected by the unqualified package name; required when `variants` is defined. |
| `variants` | Map of variant names for the `github` source. Variants inherit the package settings and share its binary names, version arguments and install paths. |
| `variants.<variant>.asset` | Override the asset template for this variant on all platforms. |
| `variants.<variant>.platforms` | Platform-specific `asset` and `executable` overrides; only platforms supported by the package are allowed. Empty values inherit the package settings. |
| `platforms` | Map of `linux-amd64` and/or `windows-amd64` entries. Omit it to support both. |
| `platforms.<platform>.asset` | Platform-specific GitHub asset template. |
| `platforms.<platform>.executable` | Relative executable path within an archive. Omit it to find a unique matching binary filename. |
| `platforms.<platform>.path` | Destination file; defaults to `{bin}/{binary}{ext}`. Must start with `{bin}/` or `{home}/`, or `{programfiles}/` for MSI installers. MSI requires an explicit path matching the installer's destination. PACE uses Go's bin directory and does not allow an override. |

Asset, checksum and archive-executable templates support `{name}`, `{binary}`, `{version}`, `{tag}`, `{os}`, `{arch}`, `{ext}` and `{variant}`. `{variant}` requires defined variants and expands to the selected variant name. `{version}` excludes the tag prefix; `{tag}` includes it. `{ext}` is `.exe` on Windows and empty on Linux. Destination templates additionally support `{bin}` and `{home}`, plus `{programfiles}` for MSI installers, but cannot contain version- or variant-dependent placeholders. Use forward slashes in YAML paths. For example, Bun uses `path: '{home}/.bun/bin/{binary}{ext}'`.

GitHub and PACE packages support raw executables and `.zip`, `.tar.gz` or `.tar.xz` archives containing the primary executable and any configured companions. Additional runtime files are not installed by these sources. Archives are extracted within bounded temporary directories, SHA-256 digests are verified, companions must be present and the primary binary must report the expected version (or contain matching Go build metadata for PACE) before replacing existing executables. GitHub release assets must provide a SHA-256 `digest` or a configured checksum file; Go and Zig use their official checksum metadata. Set `GITHUB_TOKEN` when authenticated GitHub API access is needed.

The `msi` source supports only `windows-amd64`, requires an `.msi` asset and uses Windows Installer for installation and removal. Tap verifies the download's SHA-256 digest before running the installer and checks the installed binary's version afterwards. Rio uses its official `rio-installer-x86_64.msi`, which manages its files, shortcuts and PATH registration.

## Generate and build

```sh
go generate ./...
go build -ldflags="-X main.Version=v1.2.3" .
```

`go generate` runs `internal/generate`, parses YAML with `github.com/goccy/go-yaml`, validates definitions and writes the deterministic `packages_gen.go` catalog. Commit the YAML and regenerated Go file together. A normal build uses that generated catalog without external package definitions. Regenerate and rebuild after editing a definition.

The CLI uses `github.com/urfave/cli/v3` and operational logging uses `github.com/coalaura/plain/minimal`.

## Releases

Push a `v*` tag, such as `v1.2.3`, to run `.github/workflows/release.yml`. It regenerates the package catalog, builds CPU-compatible Linux and Windows amd64 binaries and injects the tag into `main.Version`. Configure the `COALAURA_PFX` and `COALAURA_PFX_PASSWORD` repository secrets for signing. The workflow publishes signed `tap-linux-amd64` and `tap-windows-amd64.exe` binaries, `SHA256SUMS`, build provenance and generated release notes. Tags containing a hyphen, such as `v1.2.3-rc.1`, produce prereleases.
