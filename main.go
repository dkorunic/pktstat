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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/hako/durafmt"
	"go.uber.org/automaxprocs/maxprocs"
)

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

type StatMap map[statKey]statEntry

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

	if *timeout == 0 {
		log.Printf("Starting on interface %q using %v", *iface, captureType)
	} else {
		log.Printf("Starting on interface %q using %v, listening for %v", *iface, captureType,
			durafmt.Parse(*timeout))
	}

	statMap := make(StatMap, statsCapacity)

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
			v, ok := statMap[k.key]
			if !ok {
				v = statEntry{}
			}

			v.Size += k.size
			v.Packets++
			statMap[k.key] = v
		}
	}()

	go func(ctx context.Context) {
		runCapture(ctx, statCh, &totalBytes, &totalPackets)
	}(c1)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case s := <-signalCh:
			fmt.Fprintf(os.Stderr, "Received %v signal, trying to exit...\n", s)
			cancel()
			close(statCh)
		}
	}(c1)

	if *timeout > 0 {
		go func() {
			time.Sleep(*timeout)
			cancel()
			close(statCh)
		}()
	}

	if *interval > 0 {
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(*interval)
					outputStats(startTime, statMap, totalPackets, totalBytes)
				}
			}
		}(c1)
	}

	wg.Wait()

	outputStats(startTime, statMap, totalPackets, totalBytes)
}
