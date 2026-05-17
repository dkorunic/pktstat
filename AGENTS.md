# AGENTS.md

## Build

```sh
task              # update deps + build (default)
task build        # fmt + optimized build with PGO + version ldflags
task build-debug  # fmt + build with -race (no trimpath/PGO)
task lint         # golangci-lint v2 (5m timeout)
task fmt          # gci + gofumpt + betteralign
```

`CGO_ENABLED=1` required on non-Linux (libpcap dependency). On Darwin: `brew install libpcap` needed at build time.

Direct build: `go build -trimpath -pgo=auto -o pktstat`

Version ldflags (set in `Taskfile.yml`): `-X main.GitTag=... -X main.GitCommit=... -X main.GitDirty=... -X main.BuildTime=...`

## Linting

`.golangci.yml` (v2): `default: all` with these disabled: cyclop, depguard, dupl, exhaustruct, forbidigo, funlen, gochecknoglobals, gocognit, lll, varnamelen, wrapcheck. Run `task fmt` + `task lint` before committing.

## Architecture

Single `package main`, 7 files (300+ lines):

| File | Role |
|---|---|
| `main.go` | CLI entry, concurrency model (aggregator, capture, signal, interval goroutines), constants (`statsCapacity=131072`, `queueCapacity=8192`, `aggregateBatch=256`) |
| `capture.go` | Packet decoding loop (`DecodingLayerParser`, zero-alloc fast path), `statKey`/`statEntry`/`statChKey` types, `runCapture()` |
| `packet_linux.go` (`//go:build linux`) | AF_PACKET/TPacket v3 via `gopacket/afpacket` |
| `packet.go` (`//go:build !linux`) | PCAP (`pcap.OpenLive`) fallback |
| `flags.go` | CLI parsing (`ff/v4`) |
| `output.go` | Stats formatting and display |
| `helpers.go` | Utilities |

**Key type**: `statKey{SrcIP, DstIP netip.Addr, Proto LayerType, SrcPort, DstPort uint16}` → `StatMap` aggregator under `sync.Mutex`.

**No tests. No CI.**

## Gotchas

- Requires root or `cap_net_raw,cap_net_admin` capabilities
- macOS: libpcap headers required (not just the dylib)
- `.golangci.yml` v2 format (use `version: "2"` top-level)
- `betteralign` (from `go.uber.org/tools`/`betteralign` module) restructures for memory alignment — run `task fmt` which chains it
- `aggregator` in `main.go` uses batch draining pattern (tries to collect up to 256 keys per lock hold) — modifying locking needs care
