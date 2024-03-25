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
	jsoniter "github.com/json-iterator/go"
)

const (
	Bps  float64 = 1.0
	Kbps         = 1000 * Bps
	Mbps         = 1000 * Kbps
	Gbps         = 1000 * Mbps
	Tbps         = 1000 * Gbps
)

type statJSON struct {
	statKey
	statEntry
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// outputPlain generates output based on the statistics collected.
//
// startTime - the start time of the statistics collection.
// statMap - a map containing statistical data.
// totalPackets - total number of packets.
// totalBytes - total number of bytes.
func outputPlain(startTime time.Time, statMap *swiss.Map[statKey, statEntry], totalPackets uint64, totalBytes uint64) {
	dur := time.Since(startTime).Seconds()

	keySlice := calcBitrate(statMap, dur)

	for _, k := range keySlice {
		v, _ := statMap.Get(k)

		fmt.Printf("bitrate: %v, packets: %d, bytes: %d, proto: %v, src: %v:%v, dst: %v:%v\n",
			formatBitrate(v.Bitrate), v.Packets, v.Size, k.Proto.String(), k.SrcIP, k.SrcPort, k.DstIP, k.DstPort)
	}

	fmt.Printf("\nRead total packets: %d, total bytes: %d in %0.2f seconds\n", totalPackets, totalBytes, dur)
}

// outputJSON generates a JSON output based on the provided statistics map and time duration.
// startTime: the start time used to calculate the duration.
// statMap: the map containing statistics entries.
// totalPackets: the total number of packets.
// totalBytes: the total number of bytes.
func outputJSON(startTime time.Time, statMap *swiss.Map[statKey, statEntry], _ uint64, _ uint64) {
	dur := time.Since(startTime).Seconds()

	keySlice := calcBitrate(statMap, dur)

	statJSONs := make([]statJSON, 0, len(keySlice))

	for _, k := range keySlice {
		v, _ := statMap.Get(k)

		statJSONs = append(statJSONs, statJSON{
			statEntry: v,
			statKey:   k,
		})
	}

	out, _ := json.Marshal(statJSONs)

	fmt.Printf("%v\n", string(out))
}

// calcBitrate calculates the bitrate for each statEntry in the given statMap based on the duration provided, and returns a slice of statKeys sorted by bitrate in descending order.
//
// Parameters:
// - statMap: a map containing statKey as keys and statEntry as values
// - dur: a float64 representing the duration for bitrate calculation
// Returns:
// - []statKey: a slice of statKeys sorted by bitrate in descending order
func calcBitrate(statMap *swiss.Map[statKey, statEntry], dur float64) []statKey {
	keySlice := make([]statKey, 0, statMap.Len())

	// calculate bitrates and prepare keys for sort
	statMap.All(func(k statKey, v statEntry) bool {
		v.Bitrate = 8 * float64(v.Size) / dur

		keySlice = append(keySlice, k)
		statMap.Put(k, v)

		return true
	})

	// sort by bitrate descending
	sort.Slice(keySlice, func(i, j int) bool {
		v1, _ := statMap.Get(keySlice[i])
		v2, _ := statMap.Get(keySlice[j])

		return v1.Bitrate > v2.Bitrate
	})

	return keySlice
}

// MarshalJSON marshals the statKey struct into a JSON byte slice.
//
// It returns a byte slice containing the JSON representation of the struct
// and an error if any occurred during the marshaling process.
func (s *statKey) MarshalJSON() ([]byte, error) {
	type Alias statKey

	// create a temporary struct to override Proto field serialization
	return json.Marshal(&struct {
		*Alias
		Proto string `json:"proto"`
	}{
		Alias: (*Alias)(s),
		Proto: s.Proto.String(),
	})
}
