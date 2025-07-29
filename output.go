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

	"github.com/goccy/go-json"
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

// outputPlain generates output based on the statistics collected.
//
// startTime - the start time of the statistics collection.
// statMap - a map containing statistical data.
// totalPackets - total number of packets.
// totalBytes - total number of bytes.
func outputPlain(startTime time.Time, statMap StatMap, totalPackets, totalBytes uint64) {
	dur := time.Since(startTime).Seconds()

	keySlice := calcBitrate(statMap, dur)

	for _, k := range keySlice {
		fmt.Printf("bitrate: %v, packets: %d, bytes: %d, proto: %v, src: %v:%v, dst: %v:%v\n",
			formatBitrate(statMap[k].Bitrate), statMap[k].Packets, statMap[k].Size, k.Proto.String(), k.SrcIP, k.SrcPort, k.DstIP, k.DstPort)
	}

	fmt.Printf("\nRead total packets: %d, total bytes: %d in %0.2f seconds\n", totalPackets, totalBytes, dur)
}

// outputJSON generates a JSON output based on the provided statistics map and time duration.
// startTime: the start time used to calculate the duration.
// statMap: the map containing statistics entries.
// totalPackets: the total number of packets.
// totalBytes: the total number of bytes.
func outputJSON(startTime time.Time, statMap StatMap, _, _ uint64) {
	dur := time.Since(startTime).Seconds()

	keySlice := calcBitrate(statMap, dur)

	statJSONs := make([]statJSON, 0, len(keySlice))

	for _, k := range keySlice {
		statJSONs = append(statJSONs, statJSON{
			statEntry: statMap[k],
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
func calcBitrate(statMap StatMap, dur float64) []statKey {
	keySlice := make([]statKey, 0, len(statMap))

	// calculate bitrates and prepare keys for sort
	for k, v := range statMap {
		v.Bitrate = 8 * float64(v.Size) / dur

		keySlice = append(keySlice, k)
		statMap[k] = v
	}

	// sort by bitrate descending
	sort.Slice(keySlice, func(i, j int) bool {
		return statMap[keySlice[i]].Bitrate > statMap[keySlice[j]].Bitrate
	})

	return keySlice
}

// MarshalJSON marshals the statKey struct into a JSON byte slice.
//
// It returns a byte slice containing the JSON representation of the struct
// and an error if any occurred during the marshaling process.
func (s *statJSON) MarshalJSON() ([]byte, error) {
	type Alias statJSON
	// create a temporary struct to override Proto field serialization
	return json.Marshal(&struct {
		*Alias
		Proto string `json:"proto"`
	}{
		Alias: (*Alias)(s),
		Proto: s.Proto.String(),
	})
}

// outputStats outputs the collected statistics in either plain text or JSON format.
//
// startTime: the start time of the statistics collection.
// statMap: a map containing statistical data.
// totalPackets: total number of packets.
// totalBytes: total number of bytes.
func outputStats(startTime time.Time, statMap StatMap, totalPackets uint64, totalBytes uint64) {
	if *jsonOutput {
		outputJSON(startTime, statMap, totalPackets, totalBytes)
	} else {
		outputPlain(startTime, statMap, totalPackets, totalBytes)
	}
}
