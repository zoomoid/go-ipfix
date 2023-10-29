/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipfix

import (
	"context"
	"errors"
	"net"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var (
	// UDP packet size is globally limited by the packet header length field of 2^16-1.
	// However, additionally, IP data path MTU can cause UDP packets larger than the MTU
	// to be fragmented. If fragments are lost due to packet loss, the UDP packet cannot be
	// reassembled and is therefore dropped entirely. Preventing fragmentation therefore is
	// a good measure to prevent losing to many packets
	//
	// These days, MTU is most of the times in the ball park of 1500 Bytes. Subtracting IP and UDP
	// packet header lengths, as well as headers of various encapsulation formats, this then yields
	// a maximum packet size for UDP packets of 1420 (At least that is what yaf assumes).
	//
	// We patched yaf to support larger UDP packets, and can therefore use more bytes here
	// udpPacketBufferSize int = 0xFFFF, which in turn allows us to use more verbose DPI
	// with UDP transport (this previously caused a lot of unrecoverable crashes)
	UDPPacketBufferSize int = 1500

	// Number of packets being buffered in the channel. This effectively moves
	// packet buffering from UDP socket to the user space, which alleviates most
	// packet loss issues, but also drastically increases memory usage, in face of
	// 64kbytes allocated per packet.
	UDPChannelBufferSize int = 50
)

type UDPListener struct {
	bindAddr string
	packetCh chan []byte

	addr     *net.UDPAddr
	listener net.PacketConn
}

func NewUDPListener(bindAddr string) *UDPListener {
	return &UDPListener{
		bindAddr: bindAddr,
		packetCh: make(chan []byte, UDPChannelBufferSize),
	}
}

func (l *UDPListener) Listen(ctx context.Context) (err error) {
	logger := FromContext(ctx)
	// do this last such that the goroutine reading packets exits before closing the channel
	defer close(l.packetCh)
	l.addr, err = net.ResolveUDPAddr("udp", l.bindAddr)
	if err != nil {
		logger.Error(err, "failed to resolve UDP address", "addr", l.bindAddr)
		return err
	}
	listenConfig := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			controlErr := c.Control(func(fd uintptr) {
				err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				if err != nil {
					return
				}
				err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if controlErr != nil {
				err = controlErr
			}
			return err
		},
	}
	l.listener, err = listenConfig.ListenPacket(ctx, "udp", l.bindAddr)
	if err != nil {
		logger.Error(err, "failed to bind udp listener", "addr", l.addr)
	}
	defer l.listener.Close()

	var rerr error
	go func() {
		// allocate this buffer once and re-use it for each packet to read from the socket
		buffer := make([]byte, UDPPacketBufferSize)
		for {
			n, _, err := l.listener.ReadFrom(buffer)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				ErrorsTotal.Inc()
				rerr = err
				logger.Error(err, "failed to read from UDP socket")
				return
			}
			PacketsTotal.Inc()
			UDPPacketBytes.Add(float64(n))

			// allocate a smaller, trimmed to the actual packet size buffer to
			// dispose the large 2^16 byte buffer to not claim this memory forever,
			// as just handing "buffer[:n]" will NOT actually shrink the original object
			packet := make([]byte, n)
			copy(packet, buffer[:n])

			l.packetCh <- packet
		}
	}()

	logger.Info("Started UDP listener", "addr", l.bindAddr)

	<-ctx.Done()
	logger.Info("Shutting down UDP listener", "addr", l.bindAddr)

	// use error from reader goroutine if set
	err = rerr
	return
}

func (l *UDPListener) Messages() <-chan []byte {
	return l.packetCh
}

var (
	UDPPacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_packets_total",
		Help: "Total number of packets received via UDP listener",
	})
	UDPErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_errors_total",
		Help: "Total number of errors encountered in the UDP listener",
	})
	UDPPacketBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_packet_bytes",
		Help: "Total number of bytes read in the UDP listener",
	})
)
