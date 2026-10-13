# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Porthole is a password-protected Go HTTP server that handles chunked file uploads and downloads through a web interface with automatic ngrok tunnel support. Uses WebSocket as the primary transport with HTTP fallback.

## Development Commands

### Build and run
```bash
go build -o Porthole
./Porthole

# Or run directly
go run main.go

# Quick test with inline env
PASSWORD="test123" go run main.go

# Allow extra source IPs for this run (merged with TRUSTED_IPS from .env)
./Porthole --allow-ips 203.0.113.4,198.51.100.0/24
```

### Dependencies
```bash
go mod download   # install
go mod tidy       # update/clean
```

### Prerequisites
- Go 1.23+ (toolchain 1.24.3)
- ngrok: `brew install ngrok` (macOS) or https://ngrok.com/download
- `.env` file with at minimum `PASSWORD=...` (see `.env.example`)

### Testing
No tests exist yet. There is no test suite, linting config, CI/CD, or Makefile.

## Architecture

### Package Structure (`pkg/`)

| Package | Purpose |
|---------|---------|
| `auth` | Session manager (`session.go`), auth middleware (`middleware.go`), IP allowlist and CloudFront origin check (`ipfilter.go`) |
| `config` | Loads config from `.env` and environment variables. `.env` is searched in the working directory first, then next to the binary (symlinks resolved), so a symlinked binary finds its project config |
| `handlers` | All HTTP route handlers |
| `files` | File operations: upload assembly, listing, downloads, notepad storage |
| `websocket` | WebSocket handlers: `uploader.go`, `downloader.go`, `notepad.go` |
| `ui` | Go string templates for HTML pages (landing, login, main page variants) |
| `ngrok` | ngrok tunnel lifecycle (start, get URL, stop) |
| `share` | In-memory anonymous share-link tokens with per-link expiry |
| `cloudfront` | AWS CloudFront distribution creation and updates |

### Entry Point (`main.go`)

Orchestrates startup in order: config → session manager → file manager → handlers → signal handling → ngrok tunnel → CloudFront (if AWS creds present) → HTTP server → graceful shutdown.

### Routes

| Route | Method | Auth | Purpose |
|-------|--------|------|---------|
| `/` | GET | No | Landing page (ngrok browser warning bypass via JS) |
| `/login` | GET/POST | No | Login form and authentication |
| `/app` | GET | Yes | Main upload/download interface |
| `/upload` | POST | Yes | HTTP chunked upload (fallback) |
| `/ws-upload` | WS | Yes | WebSocket file upload (primary) |
| `/ws-files` | WS | Yes | WebSocket file listing |
| `/ws-download` | WS | Yes | WebSocket file download |
| `/ws-notepad` | WS | Yes | WebSocket notepad |
| `/files` | GET | Yes | JSON file list (HTTP fallback) |
| `/download/:filename` | GET | Yes | HTTP file download (fallback) |
| `/delete/:filename` | DELETE | Yes | HTTP file delete (fallback; primary is `delete` message on `/ws-files`) |
| `/share` | POST | Yes | Create anonymous share link (`{fileName, expiryMinutes}` → `{url, token, expiresAt}`) |
| `/dl/:token` | GET | No | Anonymous download: browsers get a WebSocket download page, curl/scripts get the file streamed directly (`?direct=1` forces direct) |
| `/ws-dl/:token` | WS | No | Anonymous WebSocket download — streams the token's file immediately on connect |

### Key Design Patterns

**WebSocket upload protocol**: Client sends JSON messages: `start` (filename/size) → `chunk` (base64 data, 512KB) → `complete`. Server responds with progress after each chunk.

**WebSocket download protocol**: Client sends `start` with filename → server sends `metadata` → server streams 512KB base64 chunks → `complete`.

**Dual transport**: All file operations have WebSocket (primary) and HTTP (fallback) paths. Client JavaScript auto-detects and falls back.

**UI is embedded**: HTML/CSS/JS lives as Go string constants in `pkg/ui/*.go`. Two main page variants: `mainpage_ws.go` (WebSocket-enabled) and `mainpage.go` (HTTP fallback).

**Authentication**: Single shared password, session IDs via crypto/rand (32 bytes), HTTP-only SameSite=Strict cookies, 24-hour expiry, in-memory session store.

**IP access control** (`pkg/auth/ipfilter.go`): `IPFilter.Restrict` wraps every route outermost in `main.go`, so denied requests short-circuit before auth or handler work. The allowlist is `TRUSTED_IPS` (.env) plus `--allow-ips` (CLI), both accepting bare IPs and CIDRs; an empty allowlist disables IP restriction, and a malformed entry is a fatal startup error rather than a silent fail-open. Genuinely local requests are always allowed so a bad allowlist can't lock you out locally — but note a loopback `RemoteAddr` is **not** sufficient proof of that: the ngrok agent runs on the same machine and forwards to localhost, so tunnelled requests also arrive from 127.0.0.1. `isLocalRequest` therefore requires loopback *and* the absence of `X-Forwarded-For` and `X-Porthole-Origin`. Getting this wrong silently disables the entire filter. Share routes (`/dl/`, `/ws-dl/`) use `RestrictOriginOnly` — exempt from the allowlist since recipients are off-network by design, but still barred from bypassing CloudFront.

**Client IP trust** (`pkg/auth/ipfilter.go`): CloudFront appends the viewer IP to `X-Forwarded-For` and this header is always forwarded, so no cache/origin-request-policy change is needed. `X-Forwarded-For` is always read **from the right** — each hop appends the peer it saw, so client-forged values can only sit to the left. Requests via CloudFront (identified by the origin secret) resolve to `CloudFront-Viewer-Address` if present, else the second-from-right XFF entry (ngrok appends the CloudFront edge IP after CloudFront appends the viewer). Direct requests use the rightmost entry, or `RemoteAddr`.

**CloudFront origin secret**: `CLOUDFRONT_ORIGIN_SECRET` is generated on first run and persisted to `.env`, then injected as the `X-Porthole-Origin` custom origin header alongside `ngrok-skip-browser-warning` (`setOriginHeader` in `pkg/cloudfront/cloudfront.go`). CloudFront overwrites any same-named viewer header, which is what makes it a valid proof of origin. Once a distribution is successfully created or updated, `RequireCloudFront()` engages and the raw ngrok URL returns 403 for everyone — closing the CDN-bypass hole. The secret is persisted rather than regenerated so restarts don't cause a ~5 minute propagation gap.

**ngrok bypass**: Landing page (`/`) serves JS that fetches `/login` with `ngrok-skip-browser-warning` header, then replaces page content.

**Share links**: `pkg/share` keeps an in-memory token→{filename, expiry} map (crypto/rand tokens, expiry chosen at creation time). `/dl/:token` content-negotiates: `Accept: text/html` gets `pkg/ui/sharepage.go` (downloads via `/ws-dl/:token`), anything else gets the raw file. The filename always resolves server-side from the token — the anonymous endpoints never accept a client-supplied filename.

### Important Constraints

- Files saved to the directory set by `DOWNLOADS_DIR` in `.env` (default `./downloads`, resolved to an absolute path at startup) — directory auto-created
- No database — all state is in-memory or filesystem
- Concurrent uploads to **different** filenames work; **same** filename will corrupt
- Thread-safe upload sessions via mutex
- No file size limits server-side
- Chunk sizes: 512KB (WebSocket), 5MB (HTTP upload)
- CloudFront integration is optional (enabled by AWS credentials in `.env`)
- IP restriction is off by default; it activates only once `TRUSTED_IPS` or `--allow-ips` is non-empty
- When CloudFront is active the ngrok URL serves nothing but 403s — use the CloudFront URL

### Dependencies

- `github.com/gorilla/websocket v1.5.3` — WebSocket support
- `github.com/joho/godotenv v1.5.1` — .env loading
- `github.com/aws/aws-sdk-go-v2/*` — CloudFront integration