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
	"time"
)

type DateTimeMicroseconds struct {
	value    time.Time
	seconds  uint32
	fraction float64
}

func NewDateTimeMicroseconds() DataType {
	return &DateTimeMicroseconds{}
}

func (t *DateTimeMicroseconds) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (t DateTimeMicroseconds) Type() string {
	return "dateTimeMicroseconds"
}

func (t *DateTimeMicroseconds) Value() interface{} {
	return t.value
}

func (t *DateTimeMicroseconds) SetValue(v any) DataType {
	b, ok := v.(time.Time)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	return t
}

func (t DateTimeMicroseconds) Length() uint16 {
	return t.DefaultLength()
}

func (t *DateTimeMicroseconds) DefaultLength() uint16 {
	return 8
}

func (t *DateTimeMicroseconds) Clone() DataType {
	return &DateTimeMicroseconds{
		value: t.value,
	}
}

// WithLength for DateTimeMicroseconds returns the default constructor, as time
// abstract data types are not reduced-length-encodable
func (*DateTimeMicroseconds) WithLength(length uint16) DataTypeConstructor {
	return NewDateTimeMicroseconds
}

func (t *DateTimeMicroseconds) SetLength(length uint16) DataType {
	// no-op because time types are always fixed-length
	return t
}

// IsReducedLength for DateTimeMicroseconds returns false, as time abstract data
// types are not reduced-length-encodable
func (*DateTimeMicroseconds) IsReducedLength() bool {
	return false
}

func (t *DateTimeMicroseconds) Decode(in io.Reader) (int, error) {
	b := make([]byte, t.Length())
	n, err := in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.seconds = binary.BigEndian.Uint32(b[0 : t.Length()/2])
	// reading the fractional part while also masking the lower 11 bits as per RFC 7011#6.1.9
	t.fraction = float64(binary.BigEndian.Uint32(b[t.Length()/2:t.Length()])&0xFFFFF800) / math.Pow(2, 32)
	t.value = NTPEpoch.Add(time.Duration(t.seconds) * time.Second).Add(time.Duration(t.fraction) * time.Second)
	return n, nil
}

func (t *DateTimeMicroseconds) Encode(w io.Writer) (int, error) {
	b := make([]byte, 0)

	seconds := uint32(t.value.Sub(NTPEpoch).Seconds())
	fraction := t.value.Sub(NTPEpoch).Seconds() - float64(seconds)

	b = binary.BigEndian.AppendUint32(b, seconds)
	fr := uint32(fraction*math.Pow(2, 32)) & 0xFFFFF800
	b = binary.BigEndian.AppendUint32(b, fr)
	return w.Write(b)
}

func (t *DateTimeMicroseconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *DateTimeMicroseconds) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewDateTimeMicroseconds
var _ DataType = &DateTimeMicroseconds{}
