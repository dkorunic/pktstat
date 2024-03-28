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

//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"
)

const (
	captureType = "AF_PACKET"
)

type afpacketHandle struct {
	TPacket *afpacket.TPacket
}

// newAfpacketHandle creates a new afpacketHandle.
//
// Parameters:
// - device: string representing the device.
// - snaplen: integer for the snap length.
// - block_size: integer for the block size.
// - num_blocks: integer for the number of blocks.
// - useVLAN: boolean indicating whether to use VLAN.
// - timeout: duration for the timeout.
// Returns:
// - *afpacketHandle: the newly created afpacketHandle.
// - error: an error if any.
func newAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	useVLAN bool, timeout time.Duration,
) (*afpacketHandle, error) {
	h := &afpacketHandle{}
	var err error

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptAddVLANHeader(useVLAN),
			afpacket.OptPollTimeout(timeout),
			afpacket.SocketRaw,
			afpacket.TPacketVersion3)
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptAddVLANHeader(useVLAN),
			afpacket.OptPollTimeout(timeout),
			afpacket.SocketRaw,
			afpacket.TPacketVersion3)
	}
	return h, err
}

// ZeroCopyReadPacketData reads packet data without a copy.
//
// No parameters.
// Returns a byte slice for data, gopacket.CaptureInfo for ci, and an error.
func (h *afpacketHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ZeroCopyReadPacketData()
}

// SetBPFFilter sets a BPF filter on the afpacket handle.
//
// It takes a filter string and snapshot length integer as parameters.
// Returns an error.
func (h *afpacketHandle) SetBPFFilter(filter string, snaplen int) (err error) {
	pcapBPF, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, snaplen, filter)
	if err != nil {
		return err
	}

	bpfIns := []bpf.RawInstruction{}
	for _, ins := range pcapBPF {
		bpfIns2 := bpf.RawInstruction{
			Op: ins.Code,
			Jt: ins.Jt,
			Jf: ins.Jf,
			K:  ins.K,
		}
		bpfIns = append(bpfIns, bpfIns2)
	}
	if h.TPacket.SetBPF(bpfIns); err != nil {
		return err
	}

	return h.TPacket.SetBPF(bpfIns)
}

// LinkType returns the Ethernet link type.
//
// No parameters.
// Returns layers.LinkType.
func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close closes the afpacketHandle.
//
// No parameters.
func (h *afpacketHandle) Close() {
	h.TPacket.Close()
}

// SocketStats returns the socket statistics for the afpacketHandle.
//
// No parameters.
// Returns afpacket.SocketStats, afpacket.SocketStatsV3, and error.
func (h *afpacketHandle) SocketStats() (as afpacket.SocketStats, asv afpacket.SocketStatsV3, err error) {
	return h.TPacket.SocketStats()
}

// SocketStats returns the socket statistics of the afpacket handle.
//
// None.
// afpacket.SocketStats, afpacket.SocketStatsV3, error.
// afpacketComputeSize calculates the frame size, block size, and number of blocks based on the provided target size in MB, snaplen, and page size.
//
// Parameters: targetSizeMb int, snaplen int, pageSize int.
// Returns: frameSize int, blockSize int, numBlocks int, err error.
func afpacketComputeSize(targetSizeMb int, snaplen int, pageSize int) (
	frameSize int, blockSize int, numBlocks int, err error,
) {
	if snaplen < pageSize {
		frameSize = pageSize / (pageSize / snaplen)
	} else {
		frameSize = (snaplen/pageSize + 1) * pageSize
	}

	// 128 is the default from the gopacket library so just use that
	blockSize = frameSize * 128
	numBlocks = (targetSizeMb * 1024 * 1024) / blockSize

	if numBlocks == 0 {
		return 0, 0, 0, fmt.Errorf("Interface buffersize is too small")
	}

	return frameSize, blockSize, numBlocks, nil
}

// initCapture initializes and configures an afpacketHandle for packet capturing.
//
// Parameters:
//
//	iface string - the interface name
//	snaplen int - the snapshot length
//	bufferSize int - the buffer size
//	filter string - the BPF filter
//	addVLAN bool - whether to add VLAN tag
//
// Returns:
//
//	*afpacketHandle - the initialized afpacketHandle for packet capturing
func initCapture(iface string, snaplen int, bufferSize int, filter string, addVLAN bool) *afpacketHandle {
	szFrame, szBlock, numBlocks, err := afpacketComputeSize(bufferSize, snaplen, os.Getpagesize())
	if err != nil {
		log.Fatal(err)
	}

	afpacketHandle, err := newAfpacketHandle(iface, szFrame, szBlock, numBlocks, addVLAN, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	err = afpacketHandle.SetBPFFilter(filter, snaplen)
	if err != nil {
		log.Fatal(err)
	}

	return afpacketHandle
}
