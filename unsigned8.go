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
)

type Unsigned8 struct {
	value uint8
}

func NewUnsigned8() DataType {
	return &Unsigned8{}
}

func (t *Unsigned8) String() string {
	return fmt.Sprintf("%v", uint8(t.value))
}

func (*Unsigned8) Type() string {
	return "unsigned8"
}

func (t *Unsigned8) Value() interface{} {
	return t.value
}

func (t *Unsigned8) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = uint8(ty)
	case int:
		t.value = uint8(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Unsigned8) Length() uint16 {
	return t.DefaultLength()
}

func (*Unsigned8) DefaultLength() uint16 {
	return 1
}

func (t *Unsigned8) Clone() DataType {
	return &Unsigned8{
		value: t.value,
	}
}

func (*Unsigned8) WithLength(length uint16) DataTypeConstructor {
	return NewUnsigned8
}

func (t *Unsigned8) SetLength(length uint16) DataType {
	// no-op, unsigned8 is already as short as we can get
	return t
}

func (*Unsigned8) IsReducedLength() bool {
	return false
}

func (t *Unsigned8) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = uint8(b[0])
	return
}

func (t *Unsigned8) Encode(w io.Writer) (int, error) {
	b := make([]byte, 1)
	b[0] = byte(t.value)
	return w.Write(b)
}

func (t *Unsigned8) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Unsigned8) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewUnsigned8
var _ DataType = &Unsigned8{}
