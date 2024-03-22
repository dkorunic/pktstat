# pktstat

[![GitHub license](https://img.shields.io/github/license/dkorunic/pktstat)](https://github.com/dkorunic/pktstat/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/dkorunic/pktstat)](https://github.com/dkorunic/pktstat/releases/latest)

![](gopher.png)

## About

pktstat is a work in progress simple replacement for ncurses-based [pktstat](https://github.com/dleonard0/pktstat). On Linux platform it uses [AF_PACKET](https://doc.dpdk.org/guides/nics/af_packet.html), and on other platforms it uses regular PCAP live wire capture.

At the end of the execution program will display per-IP and per-protocol (IPv4, IPv6, TCP, UDP, ICMPv4 and ICMPv6) statistics sorted by per-connection bps, packets and (source-IP:port, destination-IP:port) tuples.

## Usage

```
NAME
  pktstat

FLAGS
  -?, --help               display help
  -v, --add_vlan           if true, add VLAN header
      --version            display program version
  -s, --snaplen INT        snaplen (if <= 0 uses 65535) (default: 0)
  -b, --bufsize INT        interface buffersize in MB (default: 8)
  -f, --filter STRING      BPF filter
  -i, --iface STRING       interface to read from (default: any)
  -t, --timeout DURATION   timeout for packet capture (default: 0s)
```

By default pktstat listens to all interfaces without any BPF filter. It is possible to specify interface with `--iface` and specify a BPF filter either including or excluding needed traffic, for instance `--filter "not port 22"`. Timeout `--timeout` will stop execution after a specified time, but it is also possible to interrupt program with Ctrl C, SIGTERM or SIGINT.