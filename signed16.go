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
	"encoding/json"
	"fmt"
	"io"
)

type Signed16 struct {
	value int16

	length        uint16
	reducedLength bool
}

func NewSigned16() DataType {
	return &Signed16{}
}

func (t *Signed16) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (*Signed16) Type() string {
	return "signed16"
}

func (t *Signed16) Value() interface{} {
	return t.value
}

func (t *Signed16) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = int16(ty)
	case int:
		t.value = int16(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Signed16) Length() uint16 {
	if t.length > 0 {
		return t.length
	}
	return t.DefaultLength()
}

func (*Signed16) DefaultLength() uint16 {
	return 2
}

func (t *Signed16) Clone() DataType {
	return &Signed16{
		value: t.value,
	}
}

func (t *Signed16) WithLength(length uint16) DataTypeConstructor {
	if length > 0 && length < t.DefaultLength() {
		return func() DataType {
			return &Signed16{
				length:        length,
				reducedLength: true,
			}
		}
	}
	return NewSigned16
}

func (t *Signed16) SetLength(length uint16) DataType {
	// only valid lengths for signed16 are reduced-length encodings
	if length > 0 && length < t.DefaultLength() {
		t.length = length
		t.reducedLength = true
	} else {
		t.length = t.DefaultLength()
	}
	return t
}

func (t *Signed16) IsReducedLength() bool {
	return t.reducedLength
}

func (t *Signed16) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	if !t.reducedLength {
		// fast-track
		t.value = int16(binary.BigEndian.Uint16(b))
		return
	}
	// sample MSB and pad byte array with it
	msb := b[0] >> 7
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	if msb != 0 {
		for i := uint16(0); i < offset; i++ {
			// padding loop
			c[i] = 0xFF
		}
	} // abusing golangs initialization of values with 0 here, no need for the other case
	for i := uint16(0); i < t.length; i++ {
		c[i+offset] = b[i]
	}
	t.value = int16(binary.BigEndian.Uint16(c))
	return
}

func (t *Signed16) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	if !t.reducedLength {
		binary.BigEndian.PutUint16(b, uint16(t.value))
		return w.Write(b)
	}

	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	binary.BigEndian.PutUint16(c, uint16(t.value))

	for i := uint16(0); i < t.length; i++ {
		b[i] = c[i+offset]
	}
	return w.Write(b)
}

func (t *Signed16) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Signed16) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewSigned16
var _ DataType = &Signed16{}
