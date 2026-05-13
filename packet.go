// SPDX-FileCopyrightText: 2024 Dinko Korunic
// SPDX-License-Identifier: MIT

//go:build !linux

package main

import (
	"log"

	"github.com/google/gopacket/pcap"
)

const (
	captureType = "PCAP"
)

// initCapture initializes a pcap handle with the given interface, snaplen, and filter.
//
// Parameters:
//
//	iface string - the interface to capture packets from
//	snaplen int - the snapshot length for each packet
//	filter string - the BPF filter to apply to the captured packets
//
// Return:
//
//	*pcap.Handle - the initialized pcap handle
func initCapture(iface string, snaplen, _ int, filter string, _ bool) *pcap.Handle {
	handle, err := pcap.OpenLive(iface, int32(snaplen), true, pollTimeout)
	if err != nil {
		log.Fatal(err)
	}

	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	return handle
}
