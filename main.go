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
	"sync"
	"syscall"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/cockroachdb/swiss"
	"github.com/google/gopacket"
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

type statChKey struct {
	key  statKey
	size uint64
}

const (
	statsCapacity = 8192
	queueCapacity = 2048
	maxMemRatio   = 0.9
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

	statMap := swiss.New[statKey, statEntry](statsCapacity)

	c1, cancel := context.WithCancel(context.Background())
	defer cancel()

	statCh := make(chan statChKey, queueCapacity)

	startTime := time.Now()
	totalBytes := uint64(0)
	totalPackets := uint64(0)

	var wg sync.WaitGroup

	wg.Add(1) //nolint:wsl
	go func() {
		defer wg.Done()

		for k := range statCh {
			v, ok := statMap.Get(k.key)
			if !ok {
				v = statEntry{}
			}

			v.size += k.size
			v.packets++
			statMap.Put(k.key, v)
		}
	}()

	go func(ctx context.Context) {
		runCapture(ctx, statCh, &totalBytes, &totalPackets)
	}(c1)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		s := <-signalCh
		fmt.Printf("Received %v signal, trying to exit...\n", s)
		cancel()
		close(statCh)
	}()

	if *timeout > 0 {
		go func() {
			time.Sleep(*timeout)
			cancel()
			close(statCh)
		}()
	}

	wg.Wait()

	outputStats(startTime, statMap, totalPackets, totalBytes)
}
