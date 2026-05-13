// SPDX-FileCopyrightText: 2024 Dinko Korunic
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"net/netip"
	"sync/atomic"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	layersCapacity = 10
)

type statKey struct {
	SrcIP   netip.Addr         `json:"srcIp"`
	DstIP   netip.Addr         `json:"dstIp"`
	Proto   gopacket.LayerType `json:"proto"`
	SrcPort uint16             `json:"srcPort"`
	DstPort uint16             `json:"dstPort"`
}

type statEntry struct {
	Size    uint64  `json:"size"`
	Packets uint64  `json:"packets"`
	Bitrate float64 `json:"bitrate"`
}

type statChKey struct {
	key  statKey
	size uint64
}

// runCapture captures network packets and extracts relevant statistics.
//
// ctx: context.Context for cancellation signal.
// statCh: chan<- statChKey to send statistics.
// totalBytes: *atomic.Uint64 to track total bytes processed.
// totalPackets: *atomic.Uint64 to track total packets processed.
func runCapture(ctx context.Context, statCh chan<- statChKey, totalBytes, totalPackets *atomic.Uint64) {
	handle := initCapture(*iface, *snaplen, *bufferSize, *filter, *addVLAN)

	source := gopacket.ZeroCopyPacketDataSource(handle)
	defer handle.Close()

	var (
		eth   layers.Ethernet
		ip4   layers.IPv4
		ip6   layers.IPv6
		tcp   layers.TCP
		udp   layers.UDP
		icmp4 layers.ICMPv4
		icmp6 layers.ICMPv6
	)

	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &ip6, &tcp, &udp, &icmp4, &icmp6)
	parser.IgnoreUnsupported = true

	decodedLayers := make([]gopacket.LayerType, 0, layersCapacity)

	for {
		// AF_PACKET capture on Linux or regular PCAP wire capture on everything else
		data, _, err := source.ZeroCopyReadPacketData()
		if err != nil {
			continue
		}

		err = parser.DecodeLayers(data, &decodedLayers)
		if err != nil {
			continue
		}

		var k statKey

		for _, t := range decodedLayers {
			k.Proto = t

			switch t {
			case layers.LayerTypeIPv4:
				k.SrcIP = netip.AddrFrom4([4]byte(ip4.SrcIP))
				k.DstIP = netip.AddrFrom4([4]byte(ip4.DstIP))
			case layers.LayerTypeIPv6:
				k.SrcIP = netip.AddrFrom16([16]byte(ip6.SrcIP))
				k.DstIP = netip.AddrFrom16([16]byte(ip6.DstIP))
			case layers.LayerTypeTCP:
				k.SrcPort = uint16(tcp.SrcPort)
				k.DstPort = uint16(tcp.DstPort)
			case layers.LayerTypeUDP:
				k.SrcPort = uint16(udp.SrcPort)
				k.DstPort = uint16(udp.DstPort)
			}
		}

		// non-IP traffic
		if k.Proto == layers.LayerTypeEthernet {
			continue
		}

		packetLen := uint64(len(data))

		totalBytes.Add(packetLen)
		totalPackets.Add(1)

		select {
		case <-ctx.Done():
			close(statCh)

			return
		default:
		}

		statCh <- statChKey{key: k, size: packetLen}
	}
}
