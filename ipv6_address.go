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

type IPv6Address struct {
	value net.IP
}

func NewIPv6Address() DataType {
	return &IPv6Address{}
}

func (t *IPv6Address) String() string {
	return t.value.To16().String()
}

func (t IPv6Address) Type() string {
	return "ipv6Address"
}

func (t *IPv6Address) Value() interface{} {
	return t.value
}

func (t *IPv6Address) SetValue(v any) DataType {
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

func (t IPv6Address) Length() uint16 {
	return t.DefaultLength()
}

func (t *IPv6Address) DefaultLength() uint16 {
	return 16
}

func (t *IPv6Address) Clone() DataType {
	return &IPv6Address{
		value: t.value,
	}
}

func (t *IPv6Address) WithLength(length uint16) DataTypeConstructor {
	return NewIPv6Address
}

func (t *IPv6Address) SetLength(length uint16) DataType {
	// no-op because address types are always fixed-length
	return t
}

func (*IPv6Address) IsReducedLength() bool {
	return false
}

func (t *IPv6Address) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = net.IP(b)
	return
}

func (t *IPv6Address) Encode(w io.Writer) (int, error) {
	return w.Write([]byte(t.value))
}

func (t *IPv6Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *IPv6Address) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewIPv6Address
var _ DataType = &IPv6Address{}
