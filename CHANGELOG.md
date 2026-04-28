# Changelog

## v0.11 (2026-04-28)

### Added

- `CHANGELOG.md`.

### Changed

- **Loopback IP blocking** — use `net.ParseIP` and `IsLoopback()` instead of hardcoded `"127.0.0.1"` string comparison. The check now covers all `127.0.0.0/8` and `::1` addresses at the request and CONNECT handler levels.

- **DNS resolution guard** — resolve hostnames in the request handler and block requests to domains that resolve to loopback or private IPs, giving a proper 502 response instead of a dropped connection.

- **Unified rejection messages** — all blocking paths (request handler, CONNECT handler, and DialContext) now return `"Request blocked: <host>"`.

- **DialContext now blocks loopback** — previously only private IPs were blocked at dial time; loopback addresses are now also blocked, catching domains like `localhost.localdomain` that resolve to `127.0.0.1`.

### Fixed

- Fix tests for cross-compilation and `CGO_ENABLED=0`.

### Tests

- Add tests and `make test` target.
- Update `DEVELOPMENT.md` for tests.

### Nix

- Add missing `gnumake` to `shell.nix`.

## v0.10 (2026-04-28)

### Changed

- **Switch proxy library from `abourget/goproxy` to `elazarl/goproxy`** — the upstream is actively maintained and brings HTTP/2, WebSocket support, and certificate caching. The SNI sniffing from the fork was unnecessary for this project's use case.

- **Handle SNI-aware MITM for IP-based CONNECT requests** — when a client resolves DNS locally and connects via raw IP, the TLS SNI hostname is now captured and used to generate proper ephemeral certificates.

### Added

- `DEVELOPMENT.md` with build instructions and architecture overview.

### Fixed

- Properly check `rows.Err()` after row iteration in `loadAllReqInfo`.
- Return explicit error instead of bare return in `Dial`.

### Refactored

- Replace deprecated `io/ioutil` with `os` and `io` equivalents.
- Remove deprecated `rand.Seed` call.
- Add named constants for magic numbers.
- Rename `ago` to `minutesAgo` for clarity.

### CI/Build

- Dependabot now groups updates by package ecosystem.
- Replace custom `azuwis/actions/download` with `actions/download-artifact@v8`.
- Fix release job: add missing `merge-multiple` argument for `download-artifact`.
- Use `gh` command for file change detection in CI.
- Fix GitHub Actions workflow warnings from `actionlint`.

### Nix

- Drop Nix flake (`flake.nix`).
- Move `update` command to a devshell script.
- Set `torrent-ratio` as the default package output.
- Add missing `gcc` to `shell.nix`.
- Remove redundant `-w` linker flag (`-s` implies `-w`).

## v0.9 (2025-11-29)

### Added

- Inject version info via `-ldflags -X`.

### Changed

- Nix: bump nixpkgs to nixos-25.11.
- Nix: use `pname`/`version` in package derivation.
- Nix: switch fileset to include pattern.
- Nix: use devshell.
- Switch from `golang.org/x/crypto` to current supported versions.

### CI/Build

- Run CI on tag push.
- Default to build for all platforms in `workflow_dispatch`.
- Auto-commit `vendorHash` updates via CI.
- Pin and update GitHub Actions (`checkout@v6`, `upload-artifact@v5`, `cachix/cachix-action@v15`).

## v0.8 (2024-10-11)

### Added

- Support overriding `User-Agent` header.

### Changed

- Nix: use nixos-24.05.
- Nix: format using `nixfmt-rfc-style`.
- Dependabot: enable for GitHub Actions and Go modules.

### CI/Build

- Ignore `go.mod`/`go.sum` changes on push.
- `nix run .#update` to update `vendorHash`.
- Use `nix-update` for `vendorHash` updates.

## v0.7 (2024-01-09)

### Added

- Support overriding `peer_id`.
- Reject requests to local domains and private IPs.
- Execute templates based on URL and User-Agent (HTML for browsers, plain text for others).
- Handle HTTP 304 responses for embedded static files.

### Changed

- WebUI: sort descending and sort Report column by default.
- Use one `printf` in `index.txt` template.
- Update to Go 1.17.

### Added (Nix)

- Initial Nix flake with devShell and cross-compilation support (including darwin).

### CI/Build

- GitHub Actions workflow for cross-compiling binaries.
- Upload binaries to GitHub Releases.

## v0.6 (2021-03-18)

### Added

- Configurable profiles: high/origin profiles inherit from default.
- WebUI: sortable HTML table with CSS and JavaScript.
- HTTP `ServerMux` as `NonProxyHandler`.

### Changed

- Use "B" (bytes) in format strings.

## v0.5 (2020-09-17)

### Added

- Support overriding listening port via config.

## v0.4 (2020-06-04)

### Added

- Cleanup old records in database.
- Lint fixes.

## v0.3 (2020-02-17)

### Added

- YAML config file support with example config.
- Version info output.
- 404 pages for stats requests except `/`.

### Fixed

- Check if response is nil to fix panic.
- Fix logging for init request.

### Changed

- Shorten stats output.
- Rename `getInt64` to `queryInt64`.
- Rename timestamp from log if not in TTY.

## v0.2 (2020-02-01)

### Added

- Switch to `abourget/goproxy` for SNI sniffing support.
- Use struct to store command-line arguments.

### Fixed

- Fix stats page spacing.

## v0.1 (2020-01-30)

- Initial release.
