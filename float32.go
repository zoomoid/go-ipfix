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
	"math"
)

type Float32 struct {
	value float32
}

func NewFloat32() DataType {
	return &Float32{}
}

func (t *Float32) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (*Float32) Type() string {
	return "float32"
}

func (t *Float32) Value() interface{} {
	return t.value
}

func (t *Float32) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = float32(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Float32) Length() uint16 {
	return t.DefaultLength()
}

func (*Float32) DefaultLength() uint16 {
	return 4
}

func (t *Float32) Clone() DataType {
	return &Float32{
		value: t.value,
	}
}

func (*Float32) WithLength(length uint16) DataTypeConstructor {
	return NewFloat32
}

func (t *Float32) SetLength(length uint16) DataType {
	// no-op because floats types are always fixed-length
	return t
}

func (*Float32) IsReducedLength() bool {
	return false
}

func (t *Float32) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	i := binary.BigEndian.Uint32(b)
	t.value = math.Float32frombits(i)
	return nil
}

func (t *Float32) Encode(w io.Writer) (int, error) {
	s := math.Float32bits(t.value)
	b := make([]byte, t.Length())
	binary.BigEndian.PutUint32(b, s)
	return w.Write(b)
}

func (t *Float32) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Float32) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewFloat32
var _ DataType = &Float32{}
