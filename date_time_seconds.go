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
	"time"
)

type DateTimeSeconds struct {
	value time.Time
}

func NewDateTimeSeconds() DataType {
	return &DateTimeSeconds{}
}

func (t *DateTimeSeconds) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (*DateTimeSeconds) Type() string {
	return "dateTimeSeconds"
}

func (t *DateTimeSeconds) Value() interface{} {
	return t.value
}

func (t *DateTimeSeconds) SetValue(v any) DataType {
	b, ok := v.(time.Time)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	return t
}

func (t *DateTimeSeconds) Length() uint16 {
	return t.DefaultLength()
}

func (t *DateTimeSeconds) DefaultLength() uint16 {
	return 4
}

func (t *DateTimeSeconds) Clone() DataType {
	return &DateTimeSeconds{
		value: t.value,
	}
}

// WithLength for DateTimeSeconds returns the default constructor, as time
// abstract data types are not reduced-length-encodable
func (*DateTimeSeconds) WithLength(length uint16) DataTypeConstructor {
	return NewDateTimeSeconds
}

func (t *DateTimeSeconds) SetLength(length uint16) DataType {
	// no-op because time types are always fixed-length
	return t
}

// IsReducedLength for DateTimeSeconds returns false, as time abstract data
// types are not reduced-length-encodable
func (*DateTimeSeconds) IsReducedLength() bool {
	return false
}

func (t *DateTimeSeconds) Decode(in io.Reader) (int, error) {
	b := make([]byte, t.Length())
	n, err := in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	seconds := binary.BigEndian.Uint32(b)
	t.value = time.Unix(int64(seconds), 0).UTC()
	return n, nil
}

func (t *DateTimeSeconds) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	binary.BigEndian.PutUint32(b, uint32(t.value.Unix()))
	return w.Write(b)
}

func (t *DateTimeSeconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *DateTimeSeconds) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewDateTimeSeconds
var _ DataType = &DateTimeSeconds{}
