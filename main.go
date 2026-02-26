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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/hako/durafmt"
	"go.uber.org/automaxprocs/maxprocs"
)

const (
	statsCapacity  = 131072
	queueCapacity  = 8192
	aggregateBatch = 256
	pollTimeout    = 100 * time.Millisecond
	maxMemRatio    = 0.9
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
	statMapLock := sync.Mutex{}

	c1, cancel := context.WithCancel(context.Background())
	defer cancel()

	statCh := make(chan statChKey, queueCapacity)

	startTime := time.Now()

	totalBytes := atomic.Uint64{}
	totalPackets := atomic.Uint64{}

	var wg sync.WaitGroup

	//nolint:wsl
	wg.Go(func() {
		batch := make([]statChKey, 0, aggregateBatch)

		for k := range statCh {
			batch = append(batch, k)

		drain:
			for len(batch) < aggregateBatch {
				select {
				case more, ok := <-statCh:
					if !ok {
						break drain
					}

					batch = append(batch, more)
				default:
					break drain
				}
			}

			statMapLock.Lock()

			for _, item := range batch {
				v := statMap[item.key]
				v.Size += item.size
				v.Packets++
				statMap[item.key] = v
			}

			statMapLock.Unlock()

			batch = batch[:0]
		}
	})

	wg.Go(func() {
		runCapture(c1, statCh, &totalBytes, &totalPackets)
	})

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case s := <-signalCh:
			fmt.Fprintf(os.Stderr, "Received %v signal, trying to exit...\n", s)
			cancel()
		}
	}(c1)

	if *timeout > 0 {
		time.AfterFunc(*timeout, cancel)
	}

	if *interval > 0 {
		go func(ctx context.Context) {
			ticker := time.NewTicker(*interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					statMapLock.Lock()
					outputStats(startTime, statMap, &totalPackets, &totalBytes)
					statMapLock.Unlock()
				}
			}
		}(c1)
	}

	wg.Wait()

	statMapLock.Lock()
	outputStats(startTime, statMap, &totalPackets, &totalBytes)
	statMapLock.Unlock()
}
