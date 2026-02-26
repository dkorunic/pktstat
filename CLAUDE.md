# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

This project uses [Task](https://taskfile.dev) (`task` CLI). `CGO_ENABLED=1` is required (libpcap dependency).

```sh
task              # update deps + build (default)
task build        # fmt + optimized build with PGO and version ldflags
task build-debug  # fmt + build with race detector (no trimpath/PGO)
task lint         # fmt + golangci-lint (5m timeout)
task fmt          # gci + gofumpt + betteralign
task update       # go get -u + go mod tidy
```

Direct build (bypass task):
```sh
go build -trimpath -pgo=auto -o pktstat
```

There are no tests in this project.

## Architecture

Single `package main`, small codebase (~6 files):

**Data flow:**
```
initCapture() → ZeroCopyPacketDataSource → runCapture() → statCh (chan statChKey) → aggregator goroutine → StatMap → outputStats()
```

**Platform abstraction via build tags:**
- `packet_linux.go` (`//go:build linux`): AF_PACKET/TPacket v3 capture via `gopacket/afpacket`. Wraps `afpacketHandle` which implements `ZeroCopyPacketDataSource`.
- `packet.go` (`//go:build !linux`): PCAP-based capture via `gopacket/pcap`. Falls back to `pcap.Handle`.
- Both expose `initCapture(iface, snaplen, bufferSize, filter, addVLAN)` returning a compatible handle.

**Concurrency model (main.go):**
- `wg.Go`: aggregator goroutine reads from `statCh` and accumulates into `StatMap` under a mutex.
- `wg.Go`: capture goroutine calls `runCapture()` which loops on `ZeroCopyReadPacketData()` and sends to `statCh`.
- Signal goroutine and timeout goroutine call `cancel()` to stop both.
- Interval goroutine periodically calls `outputStats()` while capture continues.

**Key types (capture.go):**
- `statKey`: 5-tuple map key `{SrcIP netip.Addr, DstIP netip.Addr, Proto LayerType, SrcPort, DstPort uint16}`
- `statEntry`: `{Size, Packets uint64, Bitrate float64}`
- `StatMap = map[statKey]statEntry`

**Packet decoding (capture.go):**
Uses `gopacket.NewDecodingLayerParser` with pre-allocated layer structs (zero-alloc fast path). Skips pure Ethernet frames (non-IP traffic). Each decoded packet creates a `statChKey` sent to the aggregator channel.

## Formatting & Linting

The project uses `golangci-lint` with `default: all` linters minus specific disabled ones (see `.golangci.yml`). Formatters: `gci`, `gofmt`, `gofumpt`, `goimports`. Always run `task fmt` before committing. `betteralign` restructures structs for optimal memory alignment.

## Version Injection

Build-time version variables are set via ldflags in `Taskfile.yml`:
```
-X main.GitTag=... -X main.GitCommit=... -X main.GitDirty=... -X main.BuildTime=...
```
These are declared in `main.go` and trimmed of whitespace in `init()`.

## Platform Notes

- On **Linux**: AF_PACKET (TPacket v3) is used — no packet loss under moderate load, no libpcap live capture overhead.
- On **Darwin/other Unix**: libpcap (`pcap.OpenLive`) is used. Requires libpcap headers at build time (`brew install libpcap` or equivalent).
- Requires root or `CAP_NET_RAW,CAP_NET_ADMIN` capabilities to capture traffic.
