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

// Boolean is the cannonic boolean data type in RFC 7011 describing boolean values.
// IPFIX encodes boolean as a single octet, where 0x01 equals true and 0x02 equal false.
// All other values should fail to decode the data type
type Boolean struct {
	value bool
}

func NewBoolean() DataType {
	return &Boolean{}
}

func (t *Boolean) String() string {
	return fmt.Sprintf("%v", bool(t.value))
}

func (Boolean) Type() string {
	return "boolean"
}

func (t *Boolean) Value() interface{} {
	return t.value
}

func (t *Boolean) SetValue(v any) DataType {
	b, ok := v.(bool)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	return t
}

func (t *Boolean) Length() uint16 {
	return t.DefaultLength()
}

func (*Boolean) DefaultLength() uint16 {
	return 1
}

func (t *Boolean) Clone() DataType {
	return &Boolean{
		value: t.value,
	}
}

// WithLength for Booleans returns the default constructor, as boolean
// abstract data types are not reduced-length encodable
func (*Boolean) WithLength(length uint16) DataTypeConstructor {
	return NewBoolean
}

func (t *Boolean) SetLength(length uint16) DataType {
	// no-op because booleans types are always fixed-length
	return t
}

// IsReducedLength for Booleans returns false, as booleans are not reduced-length-encodable
func (*Boolean) IsReducedLength() bool {
	return false
}

// Decode takes a set of bytes (specifically, SHOULD just one) and decodes it to
// a boolean information element. If in contains more than one byte, Decode panics
func (t *Boolean) Decode(in io.Reader) (int, error) {
	b := make([]byte, t.Length())
	n, err := in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	v := b[0]
	if v == 1 {
		t.value = true
	} else if v == 2 {
		t.value = false
	} else {
		return n, fmt.Errorf("failed to decode %T, %w", t, ErrIllegalDataTypeEncoding)
	}
	return n, nil
}

func (t *Boolean) Encode(w io.Writer) (int, error) {
	b := make([]byte, 1)
	if t.value {
		b[0] = byte(1) // 1 maps to true
	} else {
		b[0] = byte(2) // 2 maps to false
	}
	return w.Write(b)
}

func (t *Boolean) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Boolean) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewBoolean
var _ DataType = &Boolean{}
