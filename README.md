# tap

A small package manager for Windows and Linux (amd64). Installs and updates tools from a [built-in catalog](packages/), using their upstream releases.

## Install

### Linux

```sh
curl -fsSL https://src.ws2.sh/tap/install.sh | sh
```

- Installs to `/usr/local/bin/tap`, using `sudo` when not running as root.
- Requires `curl` and standard coreutils, including `sha256sum`.
- Prints PATH setup instructions if `/usr/local/bin` is missing from your PATH.

### Windows

```powershell
irm https://src.ws2.sh/tap/install.ps1 | iex
```

- Installs to `%USERPROFILE%\.bin\tap.exe` and adds that directory to your user and current PowerShell session PATH.
- Requires PowerShell 5.1 or newer. No administrator privileges needed to install Tap.

Both scripts download the latest stable [GitHub release](https://github.com/coalaura/tap/releases/latest) and verify it against the release's `SHA256SUMS`.

## Usage

```sh
tap find                # browse the catalog
tap find compiler       # search names and descriptions
tap get go zig bun      # install or update packages
tap get --dry go bun    # check for updates without installing
tap info go bun         # show details, installed versions and latest versions
tap ls                  # list installed catalog packages
tap rm bun              # remove a package
tap tap                 # update tap itself
```

`get`, `rm` and `info` accept multiple package names. Names are case-insensitive. Run `tap --help` for help or `tap --version` for Tap's version.

`get` installs the latest stable release and leaves an equal or newer installation of the selected variant alone. `ls` and `find` work offline; `get`, `info` and `tap tap` check remote releases.

### Reinstalling and switching variants

```sh
tap get --force bun       # reinstall the latest stable release
tap get --dry --force bun # preview a forced reinstall
tap get onino             # use the default CPU variant
tap get onino:gpu         # install or switch to the GPU variant
```

- `--force` skips the installed-version check and can replace a newer version with the latest stable release.
- Variants share an install path; switching variants replaces the installed one.
- Installed variants are recorded in `~/.config/tap.yml` on both platforms.

## Install locations

| Package | Windows | Linux |
| --- | --- | --- |
| Go | `%ProgramFiles%\Go` (official MSI) | `/usr/local/go` |
| PACE | `%ProgramFiles%\Go\bin` | `/usr/local/go/bin` |
| Zig | `%USERPROFILE%\.zig` | `/usr/local/zig` |
| Bun | `%USERPROFILE%\.bun\bin` | `~/.bun/bin` |
| Rio | `%ProgramFiles%\Rio` (official MSI) | Not supported |
| Other executables | `%USERPROFILE%\.bin` | `/usr/local/bin` |

Tap finds installed packages at these configured locations, rather than searching PATH. Add the relevant binary directories to PATH to run the tools.

- **Linux:** Tap checks write permissions and uses `sudo` when needed, with a retry if a later write fails. Dry runs and up-to-date packages do not prompt for elevation.
- **Windows:** MSI packages require administrator privileges. Tap does not automatically elevate.
- **Downloads:** SHA-256 checksums and installed versions are verified. Set `GITHUB_TOKEN` if you need authenticated GitHub API access.

### Package notes

- **PACE:** Requires the exact matching Go release. Use `tap get go pace` to update both in order. Replacing Go's directory on Linux removes PACE, so reinstall PACE afterwards if updating Go separately.
- **Companions:** `findutils` includes `find`, `locate`, `updatedb` and `xargs`; `age` includes `age-keygen`; PACE includes `compilepe` and `asmpe`. Companions are installed and removed together.
- **diffutils:** Uses a multi-call binary: `diffutils diff ...` or `diffutils cmp ...`.
- **Removal:** `rm` removes executables, or the complete toolchain directory for Go and Zig. Bun's cache, configuration and other user data remain. Removing Go or Rio on Windows downloads the matching official MSI and requires an MSI-installed distribution.

## Add a package

Create a file such as `packages/example.yml`:

```yaml
# yaml-language-server: $schema=./schema.json

description: An example command-line tool
repository: owner/example
```

The filename supplies the package and binary names. Defaults are GitHub releases, a `v` tag prefix, `--version` arguments, both platforms and assets named `{binary}-{os}-{arch}{ext}`.

For platform-specific archives, add entries such as:

```yaml
platforms:
  linux-amd64:
    asset: example-{version}-linux.tar.gz
    executable: example/bin/example
  windows-amd64:
    asset: example-{version}-windows.zip
    executable: example/bin/example.exe
```

- See [`packages/schema.json`](packages/schema.json) for all fields and [`packages/`](packages/) for examples. The schema comment enables editor validation and completion.
- GitHub packages support raw executables and `.zip`, `.tar.gz` or `.tar.xz` archives. Only the primary executable and configured companions are installed.
- Release assets must provide a SHA-256 digest or a configured checksum file. Go and Zig use official checksum metadata.
- After changes, run `go generate ./...` and commit both the YAML and regenerated `packages_gen.go`.

## Build

Use the Go version declared in [`go.mod`](go.mod):

```sh
go generate ./...
go build -ldflags="-X main.Version=v1.2.3" .
```

Generation validates the package definitions and writes the catalog used by the build. Package files are not needed at runtime. Rebuild after changing definitions; omit the linker flag to build with the version `dev`.
