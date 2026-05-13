// SPDX-FileCopyrightText: 2024 Dinko Korunic
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

const (
	defaultBufferSize = 8
	defaultBPFFilter  = ""
	defaultSnapLen    = 0
	defaultIface      = "any"
	defaultTimeout    = 10 * time.Minute
)

var (
	iface, filter                      *string
	snaplen, bufferSize                *int
	addVLAN, version, help, jsonOutput *bool
	timeout, interval                  *time.Duration
)

func parseFags() {
	fs := ff.NewFlagSet("pktstat")

	help = fs.Bool('?', "help", "display help")
	addVLAN = fs.Bool('v', "add_vlan", "if true, add VLAN header")
	jsonOutput = fs.Bool('j', "json", "if true, output in JSON format")

	version = fs.BoolLong("version", "display program version")

	snaplen = fs.Int('s', "snaplen", defaultSnapLen, "snaplen (if <= 0 uses 65535)")
	bufferSize = fs.Int('b', "bufsize", defaultBufferSize, "interface buffersize in MB")

	filter = fs.String('f', "filter", defaultBPFFilter, "BPF filter")
	iface = fs.String('i', "iface", findFirstEtherIface(), "interface to read from")

	timeout = fs.Duration('t', "timeout", defaultTimeout, "timeout for packet capture")
	interval = fs.Duration('l', "interval", 0, "interval between packet capture output")

	var err error

	if err = ff.Parse(fs, os.Args[1:]); err != nil {
		fmt.Printf("%s\n", ffhelp.Flags(fs))
		fmt.Printf("Error: %v\n", err)

		os.Exit(1)
	}

	if *help {
		fmt.Printf("%s\n", ffhelp.Flags(fs))

		os.Exit(0)
	}

	if *version {
		fmt.Printf("pktstat %v %v%v, built on: %v\n", GitTag, GitCommit, GitDirty, BuildTime)

		os.Exit(0)
	}

	if *snaplen <= 0 {
		*snaplen = 65535
	}

	if *timeout > 0 && *interval > 0 && *interval >= *timeout {
		*interval = 0
	}
}
