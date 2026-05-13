// SPDX-FileCopyrightText: 2024 Dinko Korunic
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"net"
	"strings"
)

// formatBitrate formats the given bitrate value into a string representation.
//
// It takes a float64 value representing the bitrate and returns a string.
func formatBitrate(b float64) string {
	switch {
	case b < Kbps:
		return fmt.Sprintf("%.2f bps", b)
	case b < 10*Kbps:
		return fmt.Sprintf("%.2f Kbps", b/Kbps)
	case b < 10*Mbps:
		return fmt.Sprintf("%.2f Mbps", b/Mbps)
	case b < 10*Gbps:
		return fmt.Sprintf("%.2f Gbps", b/Gbps)
	case b < 10*Tbps:
		return fmt.Sprintf("%.2f Tbps", b/Tbps)
	}

	return fmt.Sprintf("%.2fTbps", b/Tbps)
}

// findFirstEtherIface searches for the first available Ethernet interface.
//
// It iterates over all network interfaces and checks for interfaces that are up, not loopback,
// and do not contain "docker" in their name. For each valid interface, it retrieves the associated
// IP addresses and returns the name of the first interface that has a valid IPv4 or IPv6 address.
// If no suitable interface is found, it returns the default interface name.
func findFirstEtherIface() string {
	i, err := net.Interfaces()
	if err != nil {
		return defaultIface
	}

	for _, f := range i {
		if (f.Flags&net.FlagUp == 0) || (f.Flags&net.FlagLoopback) != 0 {
			continue
		}

		if strings.Contains(f.Name, "docker") {
			continue
		}

		addrs, err := f.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			var ip net.IP

			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.To16() != nil {
				return f.Name
			}
		}
	}

	return defaultIface
}
