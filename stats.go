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
	"sort"
	"time"

	"github.com/cockroachdb/swiss"
)

// outputStats generates statistics based on the provided inputs.
//
// startTime: the time when the statistics collection started
// statMap: a map containing the statistics data
// totalPackets: total number of packets processed
// totalBytes: total number of bytes processed
func outputStats(startTime time.Time, statMap *swiss.Map[statKey, statEntry], totalPackets uint64, totalBytes uint64) {
	dur := time.Since(startTime).Seconds()

	keySlice := make([]statKey, 0, statMap.Len())

	// calculate bitrates and prepare keys for sort
	statMap.All(func(k statKey, v statEntry) bool {
		v.bitrate = 8 * float64(v.size) / dur

		keySlice = append(keySlice, k)
		statMap.Put(k, v)

		return true
	})

	// sort by bitrate descending
	sort.Slice(keySlice, func(i, j int) bool {
		v1, _ := statMap.Get(keySlice[i])
		v2, _ := statMap.Get(keySlice[j])

		return v1.bitrate > v2.bitrate
	})

	for _, k := range keySlice {
		v, _ := statMap.Get(k)

		fmt.Printf("bitrate: %v, packets: %d, bytes: %d, proto: %v, src: %v:%v, dst: %v:%v\n",
			formatBitrate(v.bitrate), v.packets, v.size, k.proto, k.srcIP, k.srcPort, k.dstIP, k.dstPort)
	}

	fmt.Printf("\nRead total packets: %d, total bytes: %d in %0.2f seconds\n", totalPackets, totalBytes, dur)
}
