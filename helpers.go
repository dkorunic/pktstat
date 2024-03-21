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
	"fmt"
	"net"
	"net/netip"
)

// netip2Addr description of the Go function.
//
// Takes a net.IP parameter and returns a netip.Addr.
func netip2Addr(ip net.IP) netip.Addr {
	addr, _ := netip.AddrFromSlice(ip)

	return addr
}

// formatBitrate formats the bitrate value into a human-readable string based on different units.
//
// b: float64 - the bitrate value to be formatted
// string - the formatted bitrate string
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
