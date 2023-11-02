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
	"encoding/binary"
	"fmt"
	"io"
)

type Message struct {
	Version             uint16 `json:"version,omitempty" yaml:"version,omitempty"`
	Length              uint16 `json:"length,omitempty" yaml:"length,omitempty"`
	ExportTime          uint32 `json:"export_time,omitempty" yaml:"exportTime,omitempty"`
	SequenceNumber      uint32 `json:"sequence_number,omitempty" yaml:"sequenceNumber,omitempty"`
	ObservationDomainId uint32 `json:"observation_domain_id,omitempty" yaml:"observationDomainId,omitempty"`
	Sets                []Set  `json:"sets,omitempty" yaml:"sets,omitempty"`
}

func (p *Message) String() string {
	s := make([]string, 0, len(p.Sets))
	for _, set := range p.Sets {
		s = append(s, set.String())
	}
	return fmt.Sprintf("{version:%d length:%d exportTime:%d sequenceNumber:%d observationDomainId:%d sets:%v}",
		p.Version,
		p.Length,
		p.ExportTime,
		p.SequenceNumber,
		p.ObservationDomainId,
		s,
	)
}

func (p *Message) Encode(w io.Writer) (int, error) {
	b := make([]byte, 0)

	// packet header
	b = binary.BigEndian.AppendUint16(b, uint16(p.Version))
	b = binary.BigEndian.AppendUint16(b, p.Length)
	b = binary.BigEndian.AppendUint32(b, p.ExportTime)
	b = binary.BigEndian.AppendUint32(b, p.SequenceNumber)
	b = binary.BigEndian.AppendUint32(b, p.ObservationDomainId)

	nh, err := w.Write(b)
	if err != nil {
		return nh, err
	}

	// message payload
	var nb int
	for _, fs := range p.Sets {
		nfs, err := fs.Encode(w)
		nb += nfs
		if err != nil {
			return nb, err
		}
	}
	return nh + nb, err
}

func (p *Message) Decode(r io.Reader) (int, error) {
	var carry int = 0
	var shortbuf []byte = make([]byte, 2)
	var longbuf []byte = make([]byte, 4)

	n, err := r.Read(shortbuf)
	carry += n
	if err != nil {
		return carry, err
	}
	p.Version = binary.BigEndian.Uint16(shortbuf)

	if p.Version != 10 {
		return carry, UnknownVersion(p.Version)
	}

	n, err = r.Read(shortbuf)
	carry += n
	if err != nil {
		return 0, err
	}
	p.Length = binary.BigEndian.Uint16(shortbuf)

	n, err = r.Read(longbuf)
	carry += n
	if err != nil {
		return carry, err
	}
	p.ExportTime = binary.BigEndian.Uint32(longbuf)

	n, err = r.Read(longbuf)
	carry += n
	if err != nil {
		return carry, err
	}
	p.SequenceNumber = binary.BigEndian.Uint32(longbuf)

	n, err = r.Read(longbuf)
	carry += n
	if err != nil {
		return carry, err
	}
	p.ObservationDomainId = binary.BigEndian.Uint32(longbuf)

	return carry, nil
}
