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

type Signed64 struct {
	value int64

	reducedLength bool
	length        uint16
}

func NewSigned64() DataType {
	return &Signed64{}
}

func (t *Signed64) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (*Signed64) Type() string {
	return "signed64"
}

func (t *Signed64) Value() interface{} {
	return t.value
}

func (t *Signed64) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = int64(ty)
	case int:
		t.value = int64(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Signed64) Length() uint16 {
	if t.length > 0 {
		return t.length
	}
	return t.DefaultLength()
}

func (*Signed64) DefaultLength() uint16 {
	return 8
}

func (t *Signed64) Clone() DataType {
	return &Signed64{
		value: t.value,
	}
}

func (t *Signed64) WithLength(length uint16) DataTypeConstructor {
	if length > 0 && length < t.DefaultLength() {
		return func() DataType {
			return &Signed64{
				length:        length,
				reducedLength: true,
			}
		}
	}
	return NewSigned64
}

func (t *Signed64) SetLength(length uint16) DataType {
	// only valid lengths for signed64 are reduced-length encodings
	if length > 0 && length < t.DefaultLength() {
		t.length = length
		t.reducedLength = true
	} else {
		t.length = t.DefaultLength()
	}
	return t
}

func (t *Signed64) IsReducedLength() bool {
	return t.reducedLength
}

func (t *Signed64) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	if !t.reducedLength {
		// fast-track
		t.value = int64(binary.BigEndian.Uint64(b))
		return nil
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
	t.value = int64(binary.BigEndian.Uint64(c))
	return nil
}

func (t *Signed64) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	if !t.reducedLength {
		binary.BigEndian.PutUint64(b, uint64(t.value))
		return w.Write(b)
	}

	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	binary.BigEndian.PutUint64(c, uint64(t.value))

	for i := uint16(0); i < t.length; i++ {
		b[i] = c[i+offset]
	}
	return w.Write(b)
}

func (t *Signed64) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Signed64) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewSigned64
var _ DataType = &Signed64{}
