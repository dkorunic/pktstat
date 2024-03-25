// @license
// Copyright (C) 2024  Dinko Korunic
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"net/netip"

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
// totalBytes: *uint64 to track total bytes processed.
// totalPackets: *uint64 to track total packets processed.
func runCapture(ctx context.Context, statCh chan<- statChKey, totalBytes *uint64, totalPackets *uint64) {
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
		select {
		case <-ctx.Done():
			return
		default:
		}

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
				k.SrcIP = netip2Addr(ip4.SrcIP)
				k.DstIP = netip2Addr(ip4.DstIP)
			case layers.LayerTypeIPv6:
				k.SrcIP = netip2Addr(ip6.SrcIP)
				k.DstIP = netip2Addr(ip6.DstIP)
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

		*totalBytes += packetLen
		*totalPackets++

		select {
		case <-ctx.Done():
			return
		default:
		}

		statCh <- statChKey{key: k, size: packetLen}
	}
}
