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
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type IPv4Address struct {
	value net.IP
}

func NewIPv4Address() DataType {
	return &IPv4Address{}
}

func (t *IPv4Address) String() string {
	return t.value.To4().String()
}

func (*IPv4Address) Type() string {
	return "ipv4Address"
}

func (t *IPv4Address) Value() interface{} {
	return t.value
}

func (t *IPv4Address) SetValue(v any) DataType {
	switch b := v.(type) {
	case string:
		t.value = net.ParseIP(b)
	case net.IP:
		t.value = b
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T in %T", v, t.value, t))
	}
	return t
}

func (t *IPv4Address) Length() uint16 {
	return t.DefaultLength()
}

func (*IPv4Address) DefaultLength() uint16 {
	return 4
}

func (t *IPv4Address) Clone() DataType {
	return &IPv4Address{
		value: t.value,
	}
}

func (*IPv4Address) WithLength(length uint16) DataTypeConstructor {
	return NewIPv4Address
}

func (t *IPv4Address) SetLength(length uint16) DataType {
	// no-op because address types are always fixed-length
	return t
}

func (*IPv4Address) IsReducedLength() bool {
	return false
}

func (t *IPv4Address) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = net.IP(b)
	return
}

func (t *IPv4Address) Encode(w io.Writer) (int, error) {
	return w.Write([]byte(t.value))
}

func (t *IPv4Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *IPv4Address) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewIPv4Address
var _ DataType = &IPv4Address{}
