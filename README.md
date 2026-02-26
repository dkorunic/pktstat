# pktstat

[![GitHub license](https://img.shields.io/github/license/dkorunic/pktstat)](https://github.com/dkorunic/pktstat/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/dkorunic/pktstat)](https://github.com/dkorunic/pktstat/releases/latest)

![](gopher.png)

## About

pktstat is a lightweight replacement for the ncurses-based [pktstat](https://github.com/dleonard0/pktstat). On Linux it uses [AF_PACKET](https://doc.dpdk.org/guides/nics/af_packet.html); on all other platforms it uses generic live PCAP capture. It requires no special or recent kernel features — `AF_PACKET` has been available since Linux **v2.2** (1999) — and is fully compatible with other Unix platforms such as Darwin, where it falls back to generic PCAP.

At the end of execution, the program displays per-IP and per-protocol statistics (IPv4, IPv6, TCP, UDP, ICMPv4, and ICMPv6), sorted by per-connection bps, packet count, and (source-IP:port → destination-IP:port) tuples.

> **Note:** pktstat with `AF_PACKET` handles up to several thousand packets per second without loss, but for higher traffic volumes consider the [pktstat-bpf](https://github.com/dkorunic/pktstat-bpf) alternative. It is implemented as a Linux eBPF program, operates near wire-speed, and has no measurable impact on production systems.

![Demo](demo.gif)

## Requirements

Capturing traffic typically requires root privileges. As an alternative, you can run pktstat as a regular user after granting it the necessary Linux [capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html):

```shell
$ setcap cap_net_raw,cap_net_admin=eip pktstat
```

## Usage

```shell
➜ ./pktstat --help
NAME
  pktstat

FLAGS
  -?, --help                display help
  -v, --add_vlan            if true, add VLAN header
  -j, --json                if true, output in JSON format
      --version             display program version
  -s, --snaplen INT         snaplen (if <= 0 uses 65535) (default: 0)
  -b, --bufsize INT         interface buffersize in MB (default: 8)
  -f, --filter STRING       BPF filter
  -i, --iface STRING        interface to read from (default: en0)
  -t, --timeout DURATION    timeout for packet capture (default: 10m0s)
  -l, --interval DURATION   interval between packet capture output (default: 0s)
```

By default, pktstat listens on all interfaces with no BPF filter applied. Use `--iface` to select a specific interface, and `--filter` to restrict captured traffic — for example, `--filter "not port 22"` to exclude SSH traffic.

`--timeout` stops the capture after the specified duration. You can also interrupt the program at any time with Ctrl-C, SIGTERM, or SIGINT.

`--json` outputs the traffic statistics as JSON instead of plain text.

`--interval`, when set to a value greater than zero and less than the timeout, causes the program to print statistics at that cadence until it exits.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=dkorunic/pktstat,dkorunic/pktstat-bpf&type=Date)](https://star-history.com/#dkorunic/pktstat&dkorunic/pktstat-bpf&Date)
