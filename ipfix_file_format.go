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
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

type ipfixFileReader struct {
	handle io.ReadCloser

	messageCh chan []byte
	errorCh   chan error

	closer *sync.Once
}

type RawMessage []byte

// ReadFull consumes an entire io.Reader of a file containing IPFIX File Format messages
// and returns those messages as byte slices sliced up in a wrapping slice.
// Different to the IPFIXFileReader implementation this is generally synchronous.
//
// Notably, if an error occurs during reading of messages, where err != io.EOF,
// the returned []RawMessage slice is nil, while the error propagates the returned error.
// This is different to the behaviour of the IPFIXFileReader implementation.
// Additionally, io.EOF errors are NOT propagated, i.e., ReadFull just returns the []RawMessage slice
// on occurence of an EOF.
//
//	decoder := ipfix.NewDecoder(...)
//	file, _ := os.Open("flow_records.ipfix")
//	msgs, err := ipfix.ReadFull(file)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, msg := range msgs {
//		n, err := decoder.Decode(bytes.NewBuffer(msg))
//		if err != nil {
//			log.Fatal(err)
//		}
//		// Do anything with the decoded message afterwards
//	}
func ReadFull(f io.Reader) ([]RawMessage, error) {
	b := make([]RawMessage, 0)
	for {
		msg, err := readMessage(f)
		if msg != nil {
			b = append(b, msg)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return b, nil
}

// NewIPFIXFileReader creates a new reader from a file-like reader.
// It is intended to be used asynchronously using the message channel.
// from a parent context. Therefore, it needs to be started using Start(context.Context)
// inside a goroutine, because Start(context.Context) blocks until
func NewIPFIXFileReader(f io.ReadCloser) *ipfixFileReader {
	r := &ipfixFileReader{
		handle:    f,
		messageCh: make(chan []byte),
		errorCh:   make(chan error),

		closer: &sync.Once{},
	}

	return r
}

func (r *ipfixFileReader) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer r.Close()

	go func() {
		for {
			msg, err := r.readMessage()
			if msg != nil {
				r.messageCh <- msg
			}
			if err != nil {
				r.errorCh <- err
				return
			}
		}
	}()

	<-childCtx.Done()
	return nil
}

func (r *ipfixFileReader) Close() error {
	var err error
	r.closer.Do(func() {
		defer close(r.errorCh)
		defer close(r.messageCh)

		err = r.handle.Close()
	})

	return err
}

func (r *ipfixFileReader) Messages() <-chan []byte {
	return r.messageCh
}

func (r *ipfixFileReader) Errors() <-chan error {
	return r.errorCh
}

func readMessage(r io.Reader) ([]byte, error) {
	var version, length uint16

	var read int
	messageHeader := make([]byte, 4)

	n, err := r.Read(messageHeader)
	if err != nil || n != 4 {
		return nil, err
	}
	read += n

	version = binary.BigEndian.Uint16(messageHeader[0:2])
	length = binary.BigEndian.Uint16(messageHeader[2:4])

	if version != 10 {
		return nil, errors.New("ipfixFileReader: unknown protocol version number")
	}

	rem := length - 4
	payload := make([]byte, rem)
	n, err = r.Read(payload)
	read += n

	p := make([]byte, 0)
	p = append(p, messageHeader...)
	p = append(p, payload[0:n]...)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return p, err
		}
		return nil, err
	}

	return p, nil
}

func (r *ipfixFileReader) readMessage() ([]byte, error) {
	return readMessage(r.handle)
}
