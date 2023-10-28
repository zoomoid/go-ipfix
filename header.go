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
	"io"
)

const (
	// NFv9 is the NFv9 template id, kept for completeness and compatibility, the module
	// does not actually support Netflow 9 decoding out of the box.
	NFv9 uint16 = iota
	// NFv9Options is the NFv9 options template id, kept for completeness and compatibility, the module
	// does not actually support Netflow 9 decoding out of the box.
	NFv9Options
	// IPFIX is the id denoting a template set.
	IPFIX
	// IPFIXOptions is the id denoting an options template set.
	IPFIXOptions
)

type SetHeader struct {
	// 0 for TemplateSet, 1 for OptionsTemplateSet, and
	// 256-65535 for DataSet as TemplateId (thus uint16)
	Id uint16 `json:"id,omitempty"`

	Length uint16 `json:"length,omitempty"`
}

func (sh *SetHeader) Decode(r io.Reader) (n int, err error) {
	t := make([]byte, 2)
	n, err = r.Read(t)
	if err != nil {
		return
	}
	sh.Id = binary.BigEndian.Uint16(t)

	m, err := r.Read(t)
	n += m
	if err != nil {
		return
	}
	sh.Length = binary.BigEndian.Uint16(t)
	return
}

func (sh *SetHeader) Encode(w io.Writer) (n int, err error) {
	t := make([]byte, 0)

	t = binary.BigEndian.AppendUint16(t, sh.Id)
	t = binary.BigEndian.AppendUint16(t, sh.Length)

	n, err = w.Write(t)
	return
}
