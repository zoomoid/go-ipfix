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

type FixedLengthField struct {
	id uint16

	name string

	pen uint32

	reversed bool

	value DataType

	constructor DataTypeConstructor

	observationDomainId uint32

	fieldManager FieldCache

	templateManager TemplateCache

	isScope bool

	prototype *InformationElement
}

func (f *FixedLengthField) Lift() *VariableLengthField {
	return &VariableLengthField{
		id:                  f.id,
		name:                f.name,
		pen:                 f.pen,
		value:               f.value,
		fieldManager:        f.fieldManager,
		templateManager:     f.templateManager,
		constructor:         f.constructor,
		isScope:             f.isScope,
		observationDomainId: f.observationDomainId,
		prototype:           f.prototype,
	}
}

func (f *FixedLengthField) Decode(r io.Reader) (int, error) {
	if f.value == nil {
		f.value = f.constructor()
	}
	return f.value.Decode(r)
}

func (f *FixedLengthField) Encode(w io.Writer) (int, error) {
	if f.value == nil {
		return 0, nil
	}
	return f.value.Encode(w)
}

func (f *FixedLengthField) Type() string {
	if f.value == nil {
		if f.constructor == nil {
			return "undefined"
		}
		t := f.constructor()
		return t.Type()
	}
	return f.value.Type()
}

func (f *FixedLengthField) Id() uint16 {
	return f.id
}

func (f *FixedLengthField) Name() string {
	if !f.reversed {
		return f.name
	}
	return reversedName(f.name)
}

func (f *FixedLengthField) Constructor() DataTypeConstructor {
	return f.constructor
}

func (f *FixedLengthField) Prototype() *InformationElement {
	return f.prototype
}

// Value returns the fields value. If value is nil, i.e., has not yet been assigned, Value
// returns the zero value of the DataType constructor.
func (f *FixedLengthField) Value() DataType {
	if f.value == nil {
		f.value = f.constructor()
	}
	return f.value
}

func (f *FixedLengthField) SetValue(v any) Field {
	// if the value implements DataType, set the field's value directly and return
	dt, ok := v.(DataType)
	if ok {
		f.value = dt
		return f
	}

	// otherwise, call setValue on the field's data type (and constructing it first, if not done yet)
	if f.value == nil {
		f.value = f.constructor()
	}
	f.value.SetValue(v)
	return f
}

func (f *FixedLengthField) Length() uint16 {
	if f.value == nil {
		return f.constructor().Length()
	}
	return f.value.Length()
}

func (f *FixedLengthField) PEN() uint32 {
	return f.pen
}

func (f *FixedLengthField) ObservationDomainId() uint32 {
	return f.observationDomainId
}

func (f *FixedLengthField) SetScoped() Field {
	f.isScope = true
	return f
}

func (f *FixedLengthField) IsScope() bool {
	return f.isScope
}

func (f *FixedLengthField) Reversible() bool {
	return reversible(f.id)
}

func (f *FixedLengthField) Reversed() bool {
	return f.reversed
}

// consolidate converts the FixedLengthField into a format this is easily marshalled
// to JSON or other serial formats. Mainly it replaces the function component
func (f *FixedLengthField) consolidate() consolidatedField {
	pen := f.pen
	if f.reversed {
		pen = ReversePEN
	}
	cf := consolidatedField{
		Id:                  f.Id(),
		Name:                f.Name(),
		IsVariableLength:    false,
		Length:              f.Length(),
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

func (f *FixedLengthField) MarshalJSON() ([]byte, error) {
	cf := f.consolidate()
	return json.Marshal(cf)
}

func (f *FixedLengthField) UnmarshalJSON(in []byte) error {
	cf := &consolidatedField{}
	err := json.Unmarshal(in, cf)
	if err != nil {
		return err
	}
	tflf, ok := cf.restore(f.fieldManager, f.templateManager).(*FixedLengthField)
	if !ok {
		return fmt.Errorf("could not unmarshal field to variable length field")
	}
	*f = *tflf
	return nil
}

func (f *FixedLengthField) Clone() Field {
	var ndt DataType
	if dt := f.value; dt != nil {
		ndt = dt.Clone()
	}

	return &FixedLengthField{
		id:       f.id,
		name:     f.name,
		pen:      f.pen,
		isScope:  f.isScope,
		reversed: f.reversed,

		value:               ndt,
		constructor:         f.constructor,
		prototype:           f.prototype,
		observationDomainId: f.observationDomainId,
		fieldManager:        f.fieldManager,
		templateManager:     f.templateManager,
	}
}

func (f *FixedLengthField) String() string {
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

	return fmt.Sprintf("{id:%d pen:%d odid:%d name:%s length:%d variable:false reversed:%t value:%s proto:%s}",
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
