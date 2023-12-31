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

type Float64 struct {
	value float64
}

func NewFloat64() DataType {
	return &Float64{}
}

func (t *Float64) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (*Float64) Type() string {
	return "float64"
}

func (t *Float64) Value() interface{} {
	return t.value
}

func (t *Float64) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = ty
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Float64) Length() uint16 {
	return t.DefaultLength()
}

func (*Float64) DefaultLength() uint16 {
	return 8
}

func (t *Float64) Clone() DataType {
	return &Float64{
		value: t.value,
	}
}

func (*Float64) WithLength(length uint16) DataTypeConstructor {
	return NewFloat64
}

func (t *Float64) SetLength(length uint16) DataType {
	// no-op because floats types are always fixed-length
	return t
}

func (*Float64) IsReducedLength() bool {
	return false
}

func (t *Float64) Decode(in io.Reader) (int, error) {
	b := make([]byte, t.Length())
	n, err := in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	i := binary.BigEndian.Uint64(b)
	t.value = math.Float64frombits(i)
	return n, nil
}

func (t *Float64) Encode(w io.Writer) (int, error) {
	s := math.Float64bits(t.value)
	b := make([]byte, t.Length())
	binary.BigEndian.PutUint64(b, s)
	return w.Write(b)
}

func (t *Float64) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Float64) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewFloat64
var _ DataType = &Float64{}
