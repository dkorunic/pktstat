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

//go:build !linux

package main

import (
	"log"

	"github.com/google/gopacket/pcap"
)

// initCapture initializes a pcap handle with the given interface, snaplen, and filter.
//
// Parameters:
// - iface string: the interface to capture packets from
// - snaplen int: the snapshot length for captured packets
// - filter string: the BPF filter to apply to captured packets
// Return:
// *pcap.Handle: the initialized pcap handle
func initCapture(iface string, snaplen int, _ int, filter string, _ bool) *pcap.Handle {
	handle, err := pcap.OpenLive(iface, int32(snaplen), true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	return handle
}
