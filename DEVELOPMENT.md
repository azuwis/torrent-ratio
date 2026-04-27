# Development Guide

## Build / Run

```bash
make              # go build with stripped debug info
make fmt          # go fmt
make run          # run locally (127.0.0.1:8089, local config and db)
make update       # update Go dependencies
```

Nix is the primary build system:
```bash
nix-build                         # build native binary
nix-build -A linux_amd64          # cross-compile for a specific target
nix-shell                         # enter dev shell (or use direnv)
```

## Architecture

The entire application is a single-file Go program (`main.go`, ~560 lines) — a MITM proxy that intercepts BitTorrent tracker announce requests and inflates the `uploaded` query parameter to improve the client's reported ratio.

### Key types (all in `main.go`)

- **`Arg`** — CLI flags: `-addr` (listen, default `127.0.0.1:8082`), `-conf` (YAML config path), `-db` (SQLite path, default `:memory:`), `-v` (verbose), `-V` (version)
- **`Setting`** — Per-tracker ratio manipulation config: upload/download multiplier ranges, speed-based padding probability, port/peer_id/user-agent overrides
- **`ReqInfo`** — Per-torrent state stored in SQLite: info hash, reported/actual upload/download, announce epoch, incomplete count

### Flow

1. **Startup**: Parse flags → open SQLite DB → load YAML config → load MITM CA cert
2. **Request interception** (`HandleRequestFunc`): Extract `info_hash`/`uploaded`/`downloaded` from announce query params → look up host in config → optionally override peer_id/port/User-Agent → compute inflated `uploaded` value based on deltas and config → replace param → forward to real tracker
3. **Response interception** (`HandleResponseFunc`): Parse tracker response for `incomplete` (leecher count) → store in DB for future calculations
4. **Web UI**: Serve embedded `templates/` and `static/` at `/` (HTML for browsers, plain text for CLI tools) — displays a sortable table of all tracked torrents
5. **Cleanup goroutine**: Deletes entries older than `cleanupThreshold` (86400s = 1 day)
6. **Dial guard**: Custom `Transport.Dial` rejects connections to private IPs

### Key constants

- `incompleteUnknown = -2` — DB sentinel when no previous info exists for a torrent
- `maxDeltaSeconds = 10800` — skip ratio calculation if announce interval exceeds 3 hours
- `cleanupThreshold = 86400` — cleanup deletes entries older than this

### Database (SQLite)

Table `torrent` with columns: `InfoHash TEXT PRIMARY KEY`, `Host`, `ReportUploaded`, `Uploaded`, `Downloaded`, `Epoch`, `Incomplete` — all `INTEGER` except InfoHash and Host.

### Config (`torrent-ratio.yaml`)

YAML with per-tracker-hostname keys containing `Setting` values. A `"default"` key provides fallback settings. Uses YAML anchors for presets (`default`, `high`, `low`, `origin`).

### Version

`Version` variable set via `-ldflags -X main.Version=v0.9` in the Nix derivation. Defaults to `"Git"` if not set (e.g., when using `make`).

## No tests

There are zero test files in this repository. No linting configuration exists beyond `go fmt`.
