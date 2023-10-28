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
	"io"
)

type TemplateRecord struct {
	TemplateId uint16 `json:"template_id,omitempty"`
	FieldCount uint16 `json:"field_count,omitempty"`

	Fields []Field `json:"fields,omitempty"`

	TemplateManager TemplateCache `json:"-"`
	FieldManager    FieldCache    `json:"-"`
}

var _ templateRecord = &TemplateRecord{}

func (tr *TemplateRecord) Type() string {
	return KindTemplateRecord
}

func (tr *TemplateRecord) Id() uint16 {
	return tr.TemplateId
}

func (tr *TemplateRecord) Encode(w io.Writer) (n int, err error) {
	l := make([]byte, 2)
	binary.BigEndian.PutUint16(l, tr.TemplateId)
	ln, err := w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	l = make([]byte, 2)
	binary.BigEndian.PutUint16(l, tr.FieldCount)
	ln, err = w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	for _, r := range tr.Fields {
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

func (tr *TemplateRecord) Decode(r io.Reader) (n int, err error) {
	t := make([]byte, 2)
	n, err = r.Read(t)
	if err != nil {
		return
	}
	tr.TemplateId = binary.BigEndian.Uint16(t)

	m, err := r.Read(t)
	n += m
	if err != nil {
		return
	}
	tr.FieldCount = binary.BigEndian.Uint16(t)
	return
}

func (tr *TemplateRecord) DecodeData(r io.Reader) (n int, err error) {
	return
}

func (tr *TemplateRecord) MarshalJSON() ([]byte, error) {
	type iotr struct {
		TemplateId uint16 `json:"template_id,omitempty" yaml:"templateId,omitempty"`
		// FieldCount fields can be derived when reconstructing from JSON, no need to include them here...
		//
		// FieldCount      uint16 `json:"fieldCount,omitempty" yaml:"fieldCount,omitempty"`
		// ScopeFieldCount uint16 `json:"scopeFieldCount,omitempty" yaml:"scopeFieldCount,omitempty"`

		Fields []Field `json:"fields,omitempty"`
	}

	t := &iotr{
		TemplateId: tr.TemplateId,
		Fields:     tr.Fields,
	}

	return json.Marshal(t)
}

func (tr *TemplateRecord) UnmarshalJSON(in []byte) error {
	type itr struct {
		TemplateId uint16 `json:"template_id,omitempty"`
		FieldCount uint16 `json:"field_count,omitempty"`

		Fields []ConsolidatedField `json:"fields,omitempty"`
	}

	t := &itr{}
	err := json.Unmarshal(in, t)
	if err != nil {
		return err
	}
	tr.TemplateId = t.TemplateId

	// These fields are computed from the length of the fields, rather than pass-through.
	// We assume this is a bit more consistent when not needing to delimit by the length
	// tr.FieldCount = t.FieldCount
	tr.FieldCount = uint16(len(t.Fields))

	fs := make([]Field, 0, len(t.Fields))
	for _, cf := range t.Fields {
		// tr.fieldManager and tr.templateManager can still be nil
		fs = append(fs, cf.Restore(tr.FieldManager, tr.TemplateManager))
	}
	tr.Fields = fs

	return nil
}

func (tr *TemplateRecord) Length() uint16 {
	l := uint16(0)
	for _, f := range tr.Fields {
		// TemplateRecord fields do not have the intrinsic length as DataRecord fields, but rather
		// static length of sizeof(fieldId) + sizeof(fieldLength) + (penProvided ? sizeof(pen) : 0)
		// which in practice is either 4 bytes or 4+4 = 8 bytes
		if f.PEN() == 0 {
			l += 4
		} else {
			l += 8
		}
	}
	return l + 2 + 2
}
