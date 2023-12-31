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

type Signed8 struct {
	value int8
}

func NewSigned8() DataType {
	return &Signed8{}
}

func (t *Signed8) String() string {
	return fmt.Sprintf("%d", t.value)
}

func (*Signed8) Type() string {
	return "signed8"
}

func (t *Signed8) Value() interface{} {
	return t.value
}

func (t *Signed8) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = int8(ty)
	case int:
		t.value = int8(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Signed8) Length() uint16 {
	return t.DefaultLength()
}

func (t *Signed8) DefaultLength() uint16 {
	return 1
}

func (t *Signed8) Clone() DataType {
	return &Signed8{
		value: t.value,
	}
}

func (*Signed8) WithLength(length uint16) DataTypeConstructor {
	return NewSigned8
}

func (t *Signed8) SetLength(length uint16) DataType {
	// no-op, signed8 is already as short as we can get
	return t
}

func (*Signed8) IsReducedLength() bool {
	return false
}

func (t *Signed8) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = int8(uint8(b[0]))
	return
}

func (t *Signed8) Encode(w io.Writer) (int, error) {
	b := make([]byte, 1)
	b[0] = byte(uint8(t.value))
	return w.Write(b)
}

func (t *Signed8) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Signed8) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewSigned8
var _ DataType = &Signed8{}
