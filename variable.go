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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

var _ json.Marshaler = &VariableLengthField{}
var _ json.Unmarshaler = &VariableLengthField{}

type VariableLengthField struct {
	id uint16

	name string

	pen      uint32
	reversed bool

	constructor DataTypeConstructor

	value DataType

	observationDomainId uint32

	fieldManager    FieldCache
	templateManager TemplateCache

	isScope bool

	length           uint16
	longLengthFormat bool

	decoded bool

	prototype *InformationElement
}

// Variable-length fields are already encoded as such, just return the field
func (f *VariableLengthField) Lift() *VariableLengthField {
	return f
}

func (f *VariableLengthField) Type() string {
	if f.value == nil {
		if f.constructor == nil {
			return "undefined"
		}
		// don't need to call the entire *VariableLengthField.initializeValue chain here
		// we just need the type as string
		t := f.constructor()
		return t.Type()
	}
	return f.value.Type()
}

func (f *VariableLengthField) Id() uint16 {
	return f.id
}

func (f *VariableLengthField) Name() string {
	if !f.reversed {
		return f.name
	}
	return reversedName(f.name)
}

func (f *VariableLengthField) Constructor() DataTypeConstructor {
	return f.constructor
}

func (f *VariableLengthField) Prototype() *InformationElement {
	return f.prototype
}

func (f *VariableLengthField) Decode(r io.Reader) (int, error) {
	f.initializeValue()
	defer func() {
		f.decoded = true
	}()

	var length uint16
	var shortLength uint8
	var longLength uint16

	b := make([]byte, 1)
	n, err := r.Read(b)
	if err != nil {
		return n, err
	}
	shortLength = uint8(b[0])

	if shortLength == 0xFF {
		f.longLengthFormat = true
		// read two more bytes denoting a length up to 2^16 bytes
		b := make([]byte, 2)
		m, err := r.Read(b)
		n += m
		if err != nil {
			return n, err
		}
		longLength = binary.BigEndian.Uint16(b)
		length = longLength
	} else {
		length = uint16(shortLength)
	}
	f.length = length

	buf := make([]byte, length)
	m, err := r.Read(buf)
	n += m
	if err != nil {
		return n, err
	}

	// already "read" the number of bytes passed down to the DataType decoder so no need to add it again
	_, err = f.value.
		SetLength(length).           // set the decoded length here, such that the subsequent DataType level decoder consumes the right amount of bytes
		Decode(bytes.NewBuffer(buf)) // hand down a new buffer such that the parsing cannot overflow the original buffer
	return n, err
}

func (f *VariableLengthField) Encode(w io.Writer) (int, error) {
	var b []byte
	// do our own length calculation to not run into edge cases of adding static offset, exceeding the
	// long/short-form threshold and then doing the wrong decision *here*
	var length uint16
	if f.value == nil {
		// we don't know yet, so return "variable-length"
		length = 0xFFFF
	}
	ll := f.value.Length() // length of the things "in" the variable-length field
	if ll == 0 && !f.decoded {
		length = 0xFFFF
	} else {
		length = ll
	}
	if ll >= 255 || f.longLengthFormat {
		b = []byte{0xFF}
		b = binary.BigEndian.AppendUint16(b, length)
	} else {
		b = []byte{byte(uint8(length))}
	}

	n, err := w.Write(b)
	if err != nil {
		return n, err
	}
	if f.value != nil {
		nn, err := f.value.Encode(w)
		return n + nn, err
	} else {
		return n, err
	}
}

func (f *VariableLengthField) initializeValue() {
	if f.value != nil {
		// already initialized
		return
	}
	cc := NewDataTypeBuilder(f.constructor).
		SetLength(f.Length()).
		SetObservationDomain(f.observationDomainId).
		SetFieldCache(f.fieldManager).
		SetTemplateCache(f.templateManager).
		Complete()

	f.constructor = cc
	f.value = f.constructor()
}

func (f *VariableLengthField) Value() DataType {
	f.initializeValue()

	return f.value
}

func (f *VariableLengthField) Reversible() bool {
	_, nonReversible := NonReversibleFields[f.id]
	return !nonReversible
}

func (f *VariableLengthField) Reversed() bool {
	return f.reversed
}

func (f *VariableLengthField) SetValue(v any) Field {
	// if the value implements DataType, set the field's value directly and return
	dt, ok := v.(DataType)
	if ok {
		f.value = dt
		return f
	}

	// otherwise, call setValue on the field's data type (and constructing it first, if not done yet)
	f.initializeValue()

	f.value.SetValue(v)
	return f
}

func (f *VariableLengthField) Length() uint16 {
	if f.value == nil {
		// we don't know yet, so return "variable-length"
		return 0xFFFF
	}
	ll := f.value.Length() // length of the things "in" the variable-length field
	if ll == 0 && !f.decoded {
		// length for a variable-length field has not yet been determined by Decode,
		// therefore also return 0xFFFF to indicate variable-length
		return 0xFFFF
	}
	if ll >= 255 || f.longLengthFormat {
		// long-form length encoding
		return ll + 3
	} else {
		// short-form length encoding
		return ll + 1
	}
}

func (f *VariableLengthField) PEN() uint32 {
	return f.pen
}

func (f *VariableLengthField) ObservationDomainId() uint32 {
	return f.observationDomainId
}

func (f *VariableLengthField) SetScoped() Field {
	f.isScope = true
	return f
}

func (f *VariableLengthField) IsScope() bool {
	return f.isScope
}

func (f *VariableLengthField) consolidate() consolidatedField {
	pen := f.pen
	if f.reversed {
		pen = ReversePEN
	}
	cf := consolidatedField{
		Id:                  f.Id(),
		Name:                f.Name(), // this *can* include "reversed", which is then (partially) used by Restore to fully restore the semantics
		IsVariableLength:    true,
		Length:              0xFFFF,
		PEN:                 pen,
		ObservationDomainId: f.observationDomainId,
		Type:                f.Type(),
		IsScope:             f.IsScope(),
	}
	if f.value != nil {
		encValue, _ := json.Marshal(f.value)
		bb := json.RawMessage(encValue)
		cf.Value = &bb
	}
	return cf
}

func (f *VariableLengthField) MarshalJSON() ([]byte, error) {
	cf := f.consolidate()
	return json.Marshal(cf)
}

func (f *VariableLengthField) UnmarshalJSON(in []byte) error {
	cf := &consolidatedField{}
	err := json.Unmarshal(in, cf)
	if err != nil {
		return err
	}
	tvlf, ok := cf.restore(f.fieldManager, f.templateManager).(*VariableLengthField)
	if !ok {
		return fmt.Errorf("could not unmarshal field to variable length field")
	}
	*f = *tvlf
	return nil
}

func (f *VariableLengthField) Clone() Field {
	var ndt DataType
	if dt := f.value; dt != nil {
		ndt = dt.Clone()
	}

	return &VariableLengthField{
		id:                  f.id,
		name:                f.name,
		pen:                 f.pen,
		reversed:            f.reversed,
		decoded:             f.decoded,
		longLengthFormat:    f.longLengthFormat,
		length:              f.length,
		observationDomainId: f.observationDomainId,
		isScope:             f.isScope,

		value:           ndt,
		constructor:     f.constructor,
		prototype:       f.prototype,
		fieldManager:    f.fieldManager,
		templateManager: f.templateManager,
	}
}

func (f *VariableLengthField) String() string {
	val := "nil"

	if f.value != nil {
		val = fmt.Sprintf("<%s>\"%s\"", f.value.Type(), f.value.String())
	} else if f.constructor != nil {
		val = fmt.Sprintf("<%s>nil", f.constructor().Type())
	}

	proto := "nil"
	if f.prototype != nil {
		proto = f.prototype.String()
	}

	return fmt.Sprintf("{id:%d pen:%d odid:%d name:%s length:%d variable:true reversed:%t value:%s proto:%s}",
		f.id,
		f.pen,
		f.observationDomainId,
		f.name,
		f.Length(),
		f.reversed,
		val,
		proto,
	)

}
