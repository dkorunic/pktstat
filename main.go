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
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/cockroachdb/swiss"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"go.uber.org/automaxprocs/maxprocs"
)

type statKey struct {
	srcIP   netip.Addr
	dstIP   netip.Addr
	proto   gopacket.LayerType
	srcPort uint16
	dstPort uint16
}

type statEntry struct {
	size    uint64
	packets uint64
	bitrate float64
}

const (
	Bps  float64 = 1.0
	Kbps         = 1000 * Bps
	Mbps         = 1000 * Kbps
	Gbps         = 1000 * Mbps
	Tbps         = 1000 * Gbps

	statsCapacity  = 8192
	layersCapacity = 10
	maxMemRatio    = 0.9
)

var (
	GitTag    = ""
	GitCommit = ""
	GitDirty  = ""
	BuildTime = ""
)

// init initializes the GitTag, GitCommit, GitDirty, and BuildTime variables.
//
// It trims leading and trailing white spaces from the values of GitTag, GitCommit,
// GitDirty, and BuildTime.
//
//nolint:gochecknoinits
func init() {
	GitTag = strings.TrimSpace(GitTag)
	GitCommit = strings.TrimSpace(GitCommit)
	GitDirty = strings.TrimSpace(GitDirty)
	BuildTime = strings.TrimSpace(BuildTime)
}

func main() {
	parseFags()

	undo, _ := maxprocs.Set()
	defer undo()

	_, _ = memlimit.SetGoMemLimitWithOpts(
		memlimit.WithRatio(maxMemRatio),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
	)

	log.Printf("Starting on interface %q", *iface)

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

	statMap := swiss.New[statKey, statEntry](statsCapacity)

	c1, cancel := context.WithCancel(context.Background())
	defer cancel()

	exitCh := make(chan struct{})

	startTime := time.Now()
	totalBytes := uint64(0)
	totalPackets := uint64(0)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// AF-PACKET capture on Linux or regular PCAP wire capture on everything else
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
				switch t {
				case layers.LayerTypeIPv4:
					k.srcIP = netip2Addr(ip4.SrcIP)
					k.dstIP = netip2Addr(ip4.DstIP)
				case layers.LayerTypeIPv6:
					k.srcIP = netip2Addr(ip6.SrcIP)
					k.dstIP = netip2Addr(ip6.DstIP)
				case layers.LayerTypeTCP:
					k.srcPort = uint16(tcp.SrcPort)
					k.dstPort = uint16(tcp.DstPort)
					k.proto = layers.LayerTypeTCP
				case layers.LayerTypeUDP:
					k.srcPort = uint16(udp.SrcPort)
					k.dstPort = uint16(udp.DstPort)
					k.proto = layers.LayerTypeUDP
				case layers.LayerTypeICMPv4:
					k.proto = layers.LayerTypeICMPv4
				case layers.LayerTypeICMPv6:
					k.proto = layers.LayerTypeICMPv6
				}
			}

			if k.proto == 0 {
				continue
			}

			v, ok := statMap.Get(k)
			if !ok {
				v = statEntry{}
			}

			v.size += uint64(len(data))
			v.packets++
			statMap.Put(k, v)

			totalBytes += uint64(len(data))
			totalPackets++
		}
	}(c1)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case s := <-signalCh:
			fmt.Printf("Received %v signal, trying to exit...\n", s)
			cancel()
			exitCh <- struct{}{}
		}
	}()

	if *timeout > 0 {
		go func() {
			time.Sleep(*timeout)
			cancel()
			exitCh <- struct{}{}
		}()
	}

	<-exitCh

	outputStats(startTime, statMap, totalPackets, totalBytes)
}
