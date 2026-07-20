# Development Guide

## Build / Run

```bash
make              # go build with stripped debug info
make fmt          # go fmt
make test         # run tests (go test ./... -v -count=1)
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

The application is a single-file Go program (`main.go`), a MITM proxy that intercepts BitTorrent tracker announce requests and inflates the `uploaded` query parameter to improve the client's reported ratio. It uses [`elazarl/goproxy`](https://github.com/elazarl/goproxy) for the proxy core.

### Key types (all in `main.go`)

- **`Arg`**: CLI flags: `-addr` (listen, default `127.0.0.1:8082`), `-conf` (YAML config path, default `~/.torrent-ratio.yaml`), `-db` (SQLite path, default `~/.torrent-ratio.db`, falling back to `:memory:` if `$HOME` is unavailable), `-v` (verbose), `-V` (version)
- **`Setting`**: Per-tracker ratio manipulation config: upload/download multiplier ranges, speed-based padding probability, port/peer_id/user-agent overrides
- **`ReqInfo`**: Per-torrent state stored in SQLite: info hash, reported/actual upload/download, announce epoch, incomplete count

### Flow

1. **Startup**: Parse flags, open SQLite DB, load YAML config, load MITM CA cert (overrides `goproxy.GoproxyCa` directly with `tls.X509KeyPair`)
2. **CONNECT handler** (`proxy.OnRequest().HandleConnectFunc`): Reject literal loopback/private IP addresses, then MITM all CONNECT tunnels. When the target is an IP (client resolved DNS locally), the handler uses `tls.Config.GetCertificate` to capture the TLS SNI from the ClientHello, generates an ephemeral cert via `signCert()`, and stores the SNI hostname in `ctx.UserData` for inner requests.
3. **Request interception** (`OnRequest().DoFunc(makeRequestHandler(db, config))`): Reject non-FQDN, loopback IP, and hosts resolving to loopback/private IPs. Extract `info_hash`/`uploaded`/`downloaded` from announce query params. If `UserData` contains an SNI hostname, fix up `req.URL.Host` for correct upstream routing. Look up host in config. Optionally override peer_id/port/User-Agent. Compute inflated `uploaded` value based on deltas and config. Replace param. Forward to real tracker.
4. **Response interception** (`OnResponse().DoFunc(makeResponseHandler(db))`): Parse tracker response for `incomplete` (leecher count) and store in DB for future calculations.
5. **Web UI**: Serve embedded `templates/` and `static/` at `/`. HTML for browsers, plain text for CLI tools. Display a sortable table of all tracked torrents.
6. **Cleanup goroutine**: Delete entries older than `cleanupThreshold` (86400s = 1 day).
7. **Dial guard**: `Tr.DialContext = makeDialGuard()` blocks connections to loopback and private IPs.

### Key constants

- `incompleteUnknown = -2`: DB sentinel when no previous info exists for a torrent
- `maxDeltaSeconds = 10800`: skip ratio calculation if announce interval exceeds 3 hours
- `cleanupThreshold = 86400`: cleanup deletes entries older than this

### Database (SQLite)

Table `torrent` with columns: `InfoHash TEXT PRIMARY KEY`, `Host`, `ReportUploaded`, `Uploaded`, `Downloaded`, `Epoch`, `Incomplete`. All are `INTEGER` except InfoHash and Host.

### Config (`torrent-ratio.yaml`)

YAML with per-tracker-hostname keys containing `Setting` values. A `"default"` key provides fallback settings. Uses YAML anchors for presets (`default`, `high`, `low`, `origin`).

### Version

`Version` variable set via `-ldflags -X main.Version=v<version>` in the Nix derivation (run `grep 'version =' torrent-ratio.nix` to check current). Defaults to `"Git"` if not set (e.g., when using `make`).

## SNI Handling

libtorrent 2.x resolves DNS locally for HTTP proxy CONNECT, sending raw IPs
regardless of qBittorrent's "Perform hostname lookup via proxy" checkbox.
This is because libtorrent's `proxy_hostnames` setting only applies to SOCKS5
([libtorrent#7710](https://github.com/arvidn/libtorrent/pull/7710) adds a
separate `proxy_send_host_in_connect` for HTTP, disabled by default).

The proxy handles this via SNI sniffing (see Flow steps 2-3 for the mechanism).

The `signCert()` function generates ephemeral ECDSA P-256 certificates at runtime, signed by
the proxy's CA.

## Tests

Tests use a mock upstream server via `httptest`. No real network access required, so they
work in the Nix build sandbox. DB-dependent tests are skipped when `CGO_ENABLED=0`
(cross-compilation), since go-sqlite3 requires CGo.
