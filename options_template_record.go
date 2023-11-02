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
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type OptionsTemplateRecord struct {
	TemplateId      uint16 `json:"templateId,omitempty" yaml:"templateId,omitempty"`
	FieldCount      uint16 `json:"fieldCount,omitempty" yaml:"fieldCount,omitempty"`
	ScopeFieldCount uint16 `json:"scopeFieldCount,omitempty" yaml:"scopeFieldCount,omitempty"`

	Scopes  []Field `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Options []Field `json:"options,omitempty" yaml:"options,omitempty"`

	fieldCache    FieldCache
	templateCache TemplateCache
}

var _ templateRecord = &OptionsTemplateRecord{}
var _ fmt.Stringer = &OptionsTemplateRecord{}

func (otr *OptionsTemplateRecord) String() string {
	scs := make([]string, 0, len(otr.Scopes))
	for _, scope := range otr.Scopes {
		scs = append(scs, scope.String())
	}

	os := make([]string, 0, len(otr.Options))
	for _, option := range otr.Options {
		os = append(os, option.String())
	}

	return fmt.Sprintf("<id=%d,len=%d>[scopes:%v options:%v]", otr.TemplateId, otr.FieldCount, scs, os)
}

func (otr *OptionsTemplateRecord) Type() string {
	return KindOptionsTemplateSet
}

func (otr *OptionsTemplateRecord) Id() uint16 {
	return otr.TemplateId
}

func (otr *OptionsTemplateRecord) Decode(r io.Reader) (n int, err error) {
	{
		// option template record header
		t := make([]byte, 2)
		n, err = r.Read(t)
		if err != nil {
			return n, err
		}
		otr.TemplateId = binary.BigEndian.Uint16(t)

		m, err := r.Read(t)
		n += m
		if err != nil {
			return n, err
		}
		otr.FieldCount = binary.BigEndian.Uint16(t)

		m, err = r.Read(t)
		n += m
		if err != nil {
			return n, err
		}
		otr.ScopeFieldCount = binary.BigEndian.Uint16(t)

		if otr.ScopeFieldCount == 0 {
			return n, errors.New("options template record scope field count must not be zero")
		}
	}

	otr.Scopes = make([]Field, 0, int(otr.ScopeFieldCount))
	for i := 0; i < int(otr.ScopeFieldCount); i++ {
		m, err := otr.decodeScopeField(r)
		n += m
		if err != nil {
			return n, err
		}
	}

	// optionsSize is the number of fields that remain after the scopes in the Options Template record
	optionsSize := int(otr.FieldCount) - int(otr.ScopeFieldCount)
	if optionsSize < 0 {
		return n, errors.New("negative length OptionsTemplateSet")
	}
	otr.Options = make([]Field, optionsSize)
	for i := 0; i < optionsSize; i++ {
		m, err := otr.decodeOptionsField(r)
		n += m
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (otr *OptionsTemplateRecord) decodeScopeField(r io.Reader) (n int, err error) {
	f, n, err := otr.decodeTemplateField(r)
	if err != nil {
		return n, err
	}
	// TODO(zoomoid): this "should" work without reassignment because f is a pointer receiver
	f = f.SetScoped()
	otr.Scopes = append(otr.Scopes, f)
	return n, err
}

func (otr *OptionsTemplateRecord) decodeOptionsField(r io.Reader) (n int, err error) {
	f, n, err := otr.decodeTemplateField(r)
	if err != nil {
		return n, err
	}
	otr.Options = append(otr.Options, f)
	return n, err
}

func (otr *OptionsTemplateRecord) decodeTemplateField(r io.Reader) (f Field, n int, err error) {
	var rawFieldId, fieldId, fieldLength uint16
	var enterpriseId uint32
	var reverse bool

	b := make([]byte, 2)
	m, err := r.Read(b)
	n += m
	if err != nil {
		return nil, n, err
	}
	rawFieldId = binary.BigEndian.Uint16(b)

	penMask := uint16(0x8000)
	fieldId = (^penMask) & rawFieldId

	// length announcement via the template: this is either fixed or variable (i.e., 0xFFFF).
	// The FieldBuilder will therefore either create a fixed-length or variable-length field
	// on FieldBuilder.Complete()
	m, err = r.Read(b)
	n += m
	if err != nil {
		return nil, n, err
	}
	fieldLength = binary.BigEndian.Uint16(b)

	// private enterprise number parsing
	if rawFieldId >= 0x8000 {
		// first bit is 1, therefore this is a enterprise-specific IE
		b := make([]byte, 4)
		m, err := r.Read(b)
		n += m
		if err != nil {
			return nil, n, err
		}
		enterpriseId = binary.BigEndian.Uint32(b)

		if enterpriseId == ReversePEN && Reversible(fieldId) {
			reverse = true
			// clear enterprise id, because this would obscure lookup
			enterpriseId = 0
		}
	}

	fieldBuilder, err := otr.fieldCache.GetBuilder(context.TODO(), NewFieldKey(enterpriseId, fieldId))
	if err != nil {
		return nil, n, err
	}

	f = fieldBuilder.
		SetLength(fieldLength).
		SetPEN(enterpriseId).
		SetReversed(reverse).
		SetFieldManager(otr.fieldCache).
		SetTemplateManager(otr.templateCache).
		Complete()

	return f, n, nil
}

func (otr *OptionsTemplateRecord) Encode(w io.Writer) (n int, err error) {
	l := make([]byte, 2)
	binary.BigEndian.PutUint16(l, otr.TemplateId)
	ln, err := w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	l = make([]byte, 2)
	binary.BigEndian.PutUint16(l, otr.FieldCount)
	ln, err = w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	l = make([]byte, 2)
	binary.BigEndian.PutUint16(l, otr.ScopeFieldCount)
	ln, err = w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	for _, r := range otr.Scopes {
		isEnterprise := r.PEN() != 0
		b := make([]byte, 0)
		if isEnterprise {
			b = binary.BigEndian.AppendUint16(b, penMask|r.Id())
		} else {
			b = binary.BigEndian.AppendUint16(b, r.Id())
		}
		b = binary.BigEndian.AppendUint16(b, r.Length())
		if isEnterprise {
			b = binary.BigEndian.AppendUint32(b, r.PEN())
		}
		bn, err := w.Write(b)
		n += bn
		if err != nil {
			return n, err
		}
	}
	for _, r := range otr.Options {
		isEnterprise := r.PEN() != 0
		b := make([]byte, 0)
		if isEnterprise {
			b = binary.BigEndian.AppendUint16(b, penMask|r.Id())
		} else {
			b = binary.BigEndian.AppendUint16(b, r.Id())
		}
		b = binary.BigEndian.AppendUint16(b, r.Length())
		if isEnterprise {
			b = binary.BigEndian.AppendUint32(b, r.PEN())
		}
		bn, err := w.Write(b)
		n += bn
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func (otr *OptionsTemplateRecord) MarshalJSON() ([]byte, error) {
	type iotr struct {
		TemplateId uint16 `json:"template_id,omitempty" yaml:"templateId,omitempty"`
		// FieldCount fields can be derived when reconstructing from JSON, no need to include them here...
		//
		// FieldCount      uint16 `json:"fieldCount,omitempty" yaml:"fieldCount,omitempty"`
		// ScopeFieldCount uint16 `json:"scopeFieldCount,omitempty" yaml:"scopeFieldCount,omitempty"`

		Scopes  []Field `json:"scopes,omitempty" yaml:"scopes,omitempty"`
		Options []Field `json:"options,omitempty" yaml:"options,omitempty"`
	}

	t := &iotr{
		TemplateId: otr.TemplateId,
		Scopes:     otr.Scopes,
		Options:    otr.Options,
	}

	return json.Marshal(t)
}

func (otr *OptionsTemplateRecord) UnmarshalJSON(in []byte) error {
	type iotr struct {
		TemplateId      uint16 `json:"template_id,omitempty" yaml:"templateId,omitempty"`
		FieldCount      uint16 `json:"fieldCount,omitempty" yaml:"fieldCount,omitempty"`
		ScopeFieldCount uint16 `json:"scopeFieldCount,omitempty" yaml:"scopeFieldCount,omitempty"`

		Scopes  []ConsolidatedField `json:"scopes,omitempty"`
		Options []ConsolidatedField `json:"options,omitempty"`
	}

	t := &iotr{}

	err := json.Unmarshal(in, t)
	if err != nil {
		return err
	}

	otr.TemplateId = t.TemplateId

	// These fields are computed from the length of the fields, rather than pass-through.
	// We assume this is a bit more consistent when not needing to delimit by the length
	// odr.FieldCount = t.FieldCount
	// odr.ScopeFieldCount = t.ScopeFieldCount

	otr.ScopeFieldCount = uint16(len(t.Scopes))
	otr.FieldCount = uint16(len(t.Scopes) + len(t.Options))

	ss := make([]Field, 0, len(t.Scopes))
	for _, cf := range t.Scopes {
		// TODO(zoomoid): check if this is ok, i.e., "we don't need the FieldManager and TemplateManager here anymore"
		ss = append(ss, cf.Restore(otr.fieldCache, otr.templateCache))
	}
	otr.Scopes = ss

	os := make([]Field, 0, len(t.Options))
	for _, cf := range t.Scopes {
		// TODO(zoomoid): check if this is ok, i.e., "we don't need the FieldManager and TemplateManager here anymore"
		os = append(os, cf.Restore(otr.fieldCache, otr.templateCache))
	}
	otr.Options = os

	return nil
}

func (otr *OptionsTemplateRecord) Length() uint16 {
	l := uint16(0)
	for _, f := range otr.Scopes {
		// OptionsTemplateRecord fields do not have the intrinsic length as DataRecord fields, but rather
		// static length of sizeof(fieldId) + sizeof(fieldLength) + (penProvided ? sizeof(pen) : 0)
		// which in practice is either 4 bytes or 4+4 = 8 bytes
		if f.PEN() == 0 {
			l += 4
		} else {
			l += 8
		}
	}
	for _, f := range otr.Options {
		if f.PEN() == 0 {
			l += 4
		} else {
			l += 8
		}
	}
	return l + 2 + 2 + 2 // length of scopes and options + record header
}
