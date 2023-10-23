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

type Unsigned32 struct {
	value uint32

	reducedLength bool
	length        uint16
}

func NewUnsigned32() DataType {
	return &Unsigned32{}
}

var _ DataType = &Unsigned32{}

func (t *Unsigned32) String() string {
	return fmt.Sprintf("%v", uint32(t.value))
}

func (*Unsigned32) Type() string {
	return "unsigned32"
}

func (t *Unsigned32) Value() interface{} {
	return t.value
}

func (t *Unsigned32) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = uint32(ty)
	case int:
		t.value = uint32(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Unsigned32) Length() uint16 {
	if t.length > 0 {
		return t.length
	}
	return t.DefaultLength()
}

func (*Unsigned32) DefaultLength() uint16 {
	return 4
}

func (t *Unsigned32) Clone() DataType {
	return &Unsigned32{
		value: t.value,
	}
}

func (t *Unsigned32) WithLength(length uint16) DataTypeConstructor {
	if length > 0 && length < t.DefaultLength() {
		return func() DataType {
			return &Unsigned32{
				reducedLength: true,
				length:        length,
			}
		}
	}
	return NewUnsigned32
}

func (t *Unsigned32) SetLength(length uint16) DataType {
	// only valid lengths for unsigned16 are reduced-length encodings
	if length > 0 && length < t.DefaultLength() {
		t.length = length
		t.reducedLength = true
	} else {
		t.length = t.DefaultLength()
	}
	return t
}

func (t *Unsigned32) IsReducedLength() bool {
	return t.reducedLength
}

func (t *Unsigned32) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	if !t.reducedLength {
		// fast-track
		t.value = binary.BigEndian.Uint32(b)
		return nil
	}
	// because reduced-length encoding still preserves BigEndian, we pad the
	// internal uint32
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	// abusing golangs initialization of values with 0 here
	for i := uint16(0); i < t.length; i++ {
		c[i+offset] = b[i]
	}
	t.value = binary.BigEndian.Uint32(c)
	return nil
}

func (t *Unsigned32) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	if !t.reducedLength {
		// fast-track
		binary.BigEndian.PutUint32(b, t.value)
		return w.Write(b)
	}
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	binary.BigEndian.PutUint32(c, t.value)

	for i := uint16(0); i < t.length; i++ {
		b[i] = c[i+offset]
	}
	return w.Write(b)
}

func (t *Unsigned32) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Unsigned32) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewUnsigned32
var _ DataType = &Unsigned32{}
