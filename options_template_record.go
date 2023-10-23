package ipfix

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

type OptionsTemplateRecord struct {
	TemplateId      uint16 `json:"templateId,omitempty" yaml:"templateId,omitempty"`
	FieldCount      uint16 `json:"fieldCount,omitempty" yaml:"fieldCount,omitempty"`
	ScopeFieldCount uint16 `json:"scopeFieldCount,omitempty" yaml:"scopeFieldCount,omitempty"`

	Scopes  []Field `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Options []Field `json:"options,omitempty" yaml:"options,omitempty"`

	FieldManager    FieldCache    `json:"-"`
	TemplateManager TemplateCache `json:"-"`
}

var _ templateRecord = &OptionsTemplateRecord{}

func (otr *OptionsTemplateRecord) Type() string {
	return KindOptionsTemplateRecord
}

func (otr *OptionsTemplateRecord) Id() uint16 {
	return otr.TemplateId
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
		ss = append(ss, cf.Restore(otr.FieldManager, otr.TemplateManager))
	}
	otr.Scopes = ss

	os := make([]Field, 0, len(t.Options))
	for _, cf := range t.Scopes {
		// TODO(zoomoid): check if this is ok, i.e., "we don't need the FieldManager and TemplateManager here anymore"
		os = append(os, cf.Restore(otr.FieldManager, otr.TemplateManager))
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
