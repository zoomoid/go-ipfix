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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

type OctetArray struct {
	value []byte

	length uint16
}

func NewOctetArray() DataType {
	return &OctetArray{}
}

func (t *OctetArray) String() string {
	return fmt.Sprintf("%v", []byte(t.value))
}

func (*OctetArray) Type() string {
	return "octetArray"
}

func (t *OctetArray) Length() uint16 {
	// this is t.length, because we use this method in Decode, and need to set the
	// length value from outside using the Decorator WithLength, to curry the type,
	// in order to support variable-length fields.
	return t.length
}

func (t *OctetArray) Value() interface{} {
	return t.value
}

func (t *OctetArray) SetValue(v any) DataType {
	// byte arrays are base64-string encoded in JSON
	switch b := v.(type) {
	case string:
		sd, _ := base64.StdEncoding.DecodeString(b)
		t.value = sd
		t.length = uint16(len(sd))
	case []byte:
		t.value = b
		t.length = uint16(len(b))
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T in %T", v, t.value, t))
	}
	return t
}

func (*OctetArray) DefaultLength() uint16 {
	return 0
}

func (t *OctetArray) Clone() DataType {
	return &OctetArray{
		value: t.value,
	}
}

// WithLength returns a DataTypeConstructor function with a fixed, given length
func (*OctetArray) WithLength(length uint16) DataTypeConstructor {
	return func() DataType {
		return &OctetArray{
			length: length,
		}
	}
}

func (t *OctetArray) SetLength(length uint16) DataType {
	t.length = length
	return t
}

// IsReducedLength for OctetArray abstract data types returns false, as reduced-length
// encoding for arrays of bytes has no semantic value.
func (*OctetArray) IsReducedLength() bool {
	return false
}

func (t *OctetArray) Decode(in io.Reader) (n int, err error) {
	b := make([]byte, t.Length())
	n, err = in.Read(b)
	if err != nil {
		return n, fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = b
	return
}

func (t *OctetArray) Encode(w io.Writer) (int, error) {
	b := make([]byte, len(t.value))
	copy(b, t.value)
	return w.Write(b)
}

func (t *OctetArray) MarshalJSON() ([]byte, error) {
	var o string
	if t.value != nil {
		// this is the implementation where the entire byte slice is split into bytes and encoded
		// as a JSON vector containing those bytes as uint8. It is obsolete as of
		// https://laboratory.comsys.rwth-aachen.de/forensics/alexander-bartolomey/-/issues/42
		//
		// o = strings.Join(strings.Fields(fmt.Sprintf("%d", t.value)), ",")

		// this is according to the format in https://github.com/CESNET/libfds/blob/62b2f4ce11fc6e56a864bec0516bdbd32f40f7a6/src/converters/json.c#L230
		o = "0x" + hex.EncodeToString(t.value)
	} else {
		o = ""
	}
	return []byte(fmt.Sprintf("\"%s\"", o)), nil
}

// This overwrites the canonic UnmarshalJSON implementation for byte slices
func (t *OctetArray) UnmarshalJSON(in []byte) error {
	// takes in a byte slice of the form \"0x<>\" where we only want the <> part
	o, err := hex.DecodeString(string(in)[3 : len(in)-1])
	if err != nil {
		return err
	}
	t.value = o
	return nil
}

var _ DataTypeConstructor = NewOctetArray
var _ DataType = &OctetArray{}
