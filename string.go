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

type String struct {
	value string

	length uint16
}

func NewString() DataType {
	return &String{}
}

func (s *String) String() string {
	return string(s.value)
}

func (*String) Type() string {
	return "string"
}

func (t *String) Value() interface{} {
	return t.value
}

func (t *String) SetValue(v any) DataType {
	b, ok := v.(string)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	t.length = uint16(len(b))
	return t
}

func (t *String) Length() uint16 {
	// this is t.length, because we use this method in Decode, and need to set the
	// length value from outside using the Decorator WithLength, to curry the type,
	// in order to support variable-length fields.
	return t.length
}

func (*String) DefaultLength() uint16 {
	return 0
}

func (t *String) Clone() DataType {
	return &String{
		value: t.value,
	}
}

func (*String) WithLength(length uint16) DataTypeConstructor {
	return func() DataType {
		return &String{
			length: length,
		}
	}
}

func (t *String) SetLength(length uint16) DataType {
	t.length = length
	return t
}

func (*String) IsReducedLength() bool {
	return false
}

func (t *String) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	// check if in is a valid utf8 string
	// TODO(zoomoid): reactivate this, but this broke a lot of string decoding in prior versions...
	// if !utf8.Valid(b) {
	// 	// "Collecting Processes SHOULD detect and ignore such values." (RFC7011#section-6.1)
	// 	logger.V(1).Info("WARN decoded string data type that is not valid UTF-8, ignoring...", "bytes", b)
	// 	return nil
	// }
	t.value = string(b)
	return
}

func (t *String) Encode(w io.Writer) (int, error) {
	b := []byte(t.value)
	return w.Write(b)
}

func (t *String) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *String) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewString
var _ DataType = &String{}
