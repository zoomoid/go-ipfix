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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/prometheus/client_golang/prometheus"
)

type TCPListener struct {
	bindAddr string
	packetCh chan []byte

	addr     *net.TCPAddr
	listener *net.TCPListener
}

func New(bindAddr string) *TCPListener {
	return &TCPListener{
		bindAddr: bindAddr,
		packetCh: make(chan []byte, TCPChannelBufferSize),
	}
}

func (l *TCPListener) Listen(ctx context.Context) (err error) {
	logger := fromContext(ctx)

	l.addr, err = net.ResolveTCPAddr("tcp", l.bindAddr)
	if err != nil {
		return err
	}
	l.listener, err = net.ListenTCP("tcp", l.addr)
	if err != nil {
		return err
	}
	defer l.listener.Close()

	// async tcp handler function
	go func() {
		for {
			if l.listener == nil {
				return
			}
			conn, rerr := l.listener.Accept()
			TCPActiveConnections.Inc()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				ErrorsTotal.Inc()
				logger.Error(err, "failed to accept TCP connection", "addr", l.addr)
				err = rerr
				return
			}

			// handle each accepted connection in a separate goroutine for S C A L E
			// IPFIX associates an entire TCP connection with a session. It may transmit more than
			// one packet, and it may be kept alive during the entire exporting process (at least
			// that is what yaf does).
			go func(conn net.Conn) {
				if conn == nil {
					return
				}

				// initiate close after being done reading
				defer logger.V(3).Info("tcp: closed connection")
				defer TCPActiveConnections.Dec()
				defer conn.Close()

				var rerr error
				defer func() {
					if rerr != nil {
						ErrorsTotal.Inc()
					}
				}()

				// instantiate a new session from the connection to receive packets from
				session := newSessionFromConnection(conn)
				logger.V(3).Info("starting new session from TCP connection", "source", conn.RemoteAddr().String())
				errorCh := make(chan error)

				// run this loop indefinitely in a goroutine to not block. The session resets internally
				// and will be reused for subsequent packets.
				go func() {
					for {
						err := session.receive(ctx)
						if err != nil {
							errorCh <- err
							return
						}
					}
				}()

				for {
					select {
					case <-ctx.Done():
						return
					case err := <-errorCh:
						if errors.Is(err, io.EOF) {
							logger.V(1).Info("connection closed by remote", "remote_addr", conn.RemoteAddr().String())
						} else {
							logger.Error(err, "failed to read IPFIX packet", "remote_addr", conn.RemoteAddr().String())
						}
						return
					case packet := <-session.messages():
						// write packet to event source channel
						TCPReceivedBytes.Add(float64(len(packet)))
						logger.V(3).Info("wrote IPFIX packet to event source channel", "length", len(packet))
						l.packetCh <- packet
					}
				}
			}(conn)
		}
	}()

	logger.Info("Started TCP listener", "addr", l.bindAddr)

	<-ctx.Done()
	logger.Info("Shutting down TCP listener", "addr", l.addr)
	return
}

func (l *TCPListener) Messages() <-chan []byte {
	return l.packetCh
}

var (
	TCPActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tcp_listener_active_connections_total",
		Help: "Total number of active connections currently maintained by the TCP listener",
	})
	TCPErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tcp_listener_errors_total",
		Help: "Total number of errors encountered in the TCP listener",
	})
	TCPReceivedBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tcp_listener_received_bytes",
		Help: "Total number of bytes read in the TCP listener",
	})
)

var (
	TCPChannelBufferSize int = 10

	// ipfixMessageHeaderLength is the number of bytes in an IPFIX packet header
	ipfixMessageHeaderLength uint16 = 16
)

type session struct {
	offset uint16
	length uint16

	messageCh chan []byte
	message   bytes.Buffer

	reader io.Reader
}

func newSessionFromConnection(conn net.Conn) *session {
	return &session{
		messageCh: make(chan []byte),
		reader:    conn,
	}
}

func (s *session) messages() <-chan []byte {
	return s.messageCh
}

// receive successively reads from the connection's reader to piece together a message
func (s *session) receive(ctx context.Context) error {
	logger := fromContext(ctx)
	// working on header bytes
	if s.offset < ipfixMessageHeaderLength {
		_, err := s.receiveHeader()
		if err != nil {
			return err
		}

		if s.offset < ipfixMessageHeaderLength {
			// still too little bytes, will call this method again from the connection
			return nil
		}
	}

	_, err := s.receiveBody()
	if err != nil {
		return err
	}

	if s.offset < s.length {
		// not done with body yet, next call to Receive will advance the reading
		return nil
	}

	s.messageCh <- s.message.Bytes()

	// since messageCh is unbuffered, when the above unblocks, the buffer has been consumed by the handler
	// to be passed on to the EventSource. Afterwards, we can reset all internal fields for re-use
	s.length = 0
	s.offset = 0
	s.message.Reset()

	logger.V(3).Info("session: cleaning up session after received ipfix message for reuse")
	return nil
}

func (s *session) receiveHeader() (int, error) {
	var remains uint16 = ipfixMessageHeaderLength
	var offset uint16

	headerBuffer := &bytes.Buffer{}

	// if this method was already called but not enough bytes for a header were read, this is true...
	if s.offset > 0 {
		// there's already stuff in the buffer, copy it into an ipfixHeader struct
		remains = ipfixMessageHeaderLength - s.offset // update "remains" such that we don't exceed the header when reading from the socket
		_, err := headerBuffer.ReadFrom(&s.message)   // the buffer abstraction here makes it easy to do this multiple times
		if err != nil {
			return -1, fmt.Errorf("failed to read from message buffer into header buffer, %w", err)
		}
	}

	// only read "remaining bytes upto a full header"
	b := make([]byte, remains)
	len, err := s.reader.Read(b)
	if len == 0 {
		if offset > 0 {
			return len, fmt.Errorf("session closed unexpectedly: %w", err)
		}
		if errors.Is(err, io.EOF) {
			return len, err
		}
		return len, fmt.Errorf("failed to read from socket: %w", err)
	}
	_, err = headerBuffer.Write(b)
	if err != nil {
		return len, fmt.Errorf("failed to write into header buffer: %w", err)
	}

	offset += uint16(len)

	if offset < ipfixMessageHeaderLength {
		// not a whole header yet...
		// copy the internal header buffer which contains all read bytes for the header up to now
		// into the message of the session
		// reset the message buffer and write the currently read header bytes to it
		s.message.Reset()
		_, err = s.message.ReadFrom(headerBuffer)
		if err != nil {
			return -1, fmt.Errorf("failed to write from header buffer to message: %w", err)
		}
		// s.message = bytes.NewBuffer(headerBuffer.Bytes())
		s.offset = offset
		s.length = offset
		return len, nil
	}
	// now we have the full header
	b = headerBuffer.Bytes()

	// bytes 0 and 1 are "version" but we don't need that here
	msgLength := binary.BigEndian.Uint16(b[2:4])
	if err != nil {
		return len, fmt.Errorf("failed to read packet length from header buffer, %w", err)
	}
	// we've read the first 4 bytes of the headerBuffer, we need to reset the offset to be
	// able to read from it in its entirety

	// reset the entire message buffer and write the current header bytes to it
	s.message.Reset()
	_, err = s.message.Write(b)
	if err != nil {
		return len, fmt.Errorf("failed to write from header buffer to message, %w", err)
	}

	// s.message = bytes.NewBuffer(headerBuffer.Bytes())
	s.offset = ipfixMessageHeaderLength
	s.length = msgLength

	return len, nil
}

func (s *session) receiveBody() (int, error) {
	remains := s.length - s.offset

	if remains == 0 {
		// IPFIX message without a body
		return 0, nil
	}

	b := make([]byte, remains)
	len, err := s.reader.Read(b)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return len, err
		} else {
			return len, fmt.Errorf("session closed unexpectedly: %w", err)
		}
	}
	_, err = s.message.Write(b[:len]) // only write the newly read portion of the buffer to the message buffer, otherwise this adds 0s
	if err != nil {
		return len, fmt.Errorf("failed to write read bytes to message buffer, %w", err)
	}
	s.offset += uint16(len)

	return len, nil
}
