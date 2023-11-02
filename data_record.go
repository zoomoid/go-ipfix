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
	"encoding/json"
	"fmt"
	"io"
)

type DataRecord struct {
	TemplateId uint16 `json:"template_id,omitempty"`
	FieldCount uint16 `json:"field_count,omitempty"`

	Fields []Field `json:"fields,omitempty"`

	template   *Template
	fieldCache FieldCache
}

func (dr *DataRecord) Encode(w io.Writer) (n int, err error) {
	for _, r := range dr.Fields {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (dr *DataRecord) With(t *Template) *DataRecord {
	dr.template = t
	return dr
}

func (dr *DataRecord) Decode(r io.Reader) (n int, err error) {
	m := 0
	switch t := dr.template.Record.(type) {
	case *TemplateRecord:
		m, err = dr.decodeFromTempalte(r, t)
		n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, fmt.Errorf("failed to decode data set, %w", err)
		}
	case *OptionsTemplateRecord:
		m, err = dr.decodeFromOptionsTemplate(r, t)
		n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, fmt.Errorf("failed to decode data set, %w", err)
		}
	}

	ie, err := dataRecordToIE(*dr)
	if err != nil {
		return n, err
	}
	if ie != nil {
		err = dr.fieldCache.Add(context.TODO(), *ie)
		if err != nil {
			return n, err
		}
	}

	return
}

func (d *DataRecord) decodeFromTempalte(r io.Reader, t *TemplateRecord) (n int, err error) {
	m, err := d.decodeWithFields(r, t.Fields)
	n += m
	if err != nil {
		if err == io.EOF {
			return
		}
		return n, fmt.Errorf("failed to decode scope fields, %w", err)
	}
	return
}

func (d *DataRecord) decodeFromOptionsTemplate(r io.Reader, t *OptionsTemplateRecord) (n int, err error) {
	// decode all the "scope" fields first...
	n, err = d.decodeWithFields(r, t.Scopes)
	if err != nil {
		if err == io.EOF {
			return
		}
		return n, fmt.Errorf("failed to decode scope fields, %w", err)
	}
	// ...then decode all the option fields
	m := 0
	m, err = d.decodeWithFields(r, t.Options)
	n += m
	if err != nil {
		if err == io.EOF {
			return
		}
		return n, fmt.Errorf("failed to decode option fields, %w", err)
	}
	return
}

func (d *DataRecord) decodeWithFields(r io.Reader, fields []Field) (n int, err error) {
	dfs := make([]Field, 0, len(fields))
	for idx, templateField := range fields {
		// Clone the field of the template to decode the value into while also preserving the
		// template information
		tf := templateField.Clone()
		name := tf.Name()
		m, err := tf.Decode(r)
		n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, fmt.Errorf("failed to decode field (%d, %d/%d [%s]), %w", idx, tf.PEN(), tf.Id(), name, err)
		}
		dfs = append(dfs, tf)
	}
	d.Fields = dfs
	return
}

func (d *DataRecord) Length() uint16 {
	l := uint16(0)
	for _, f := range d.Fields {
		l += f.Length()
	}
	return l // header bytes are included on the Set!
}

func (dr *DataRecord) getFieldByName(enterpriseId uint32, name string) Field {
	for _, f := range dr.Fields {
		if f.PEN() == enterpriseId && f.Name() == name {
			return f
		}
	}
	return nil
}

func (dr *DataRecord) String() string {
	sl := make([]string, 0, len(dr.Fields))
	for _, v := range dr.Fields {
		sl = append(sl, v.String())
	}

	return fmt.Sprintf("<id=%d,len=%d>%v", dr.TemplateId, dr.FieldCount, sl)
}

func (dr *DataRecord) UnmarshalJSON(in []byte) error {
	type idr struct {
		TemplateId uint16 `json:"template_id,omitempty"`
		FieldCount uint16 `json:"field_count,omitempty"`

		Fields []ConsolidatedField `json:"fields,omitempty"`
	}

	t := &idr{}

	err := json.Unmarshal(in, t)
	if err != nil {
		return err
	}

	dr.TemplateId = t.TemplateId
	dr.FieldCount = t.FieldCount
	fs := make([]Field, 0, len(t.Fields))
	for _, cf := range t.Fields {
		// TODO(zoomoid): check if this is ok, i.e., "we don't need the FieldManager and TemplateManager here anymore"
		fs = append(fs, cf.Restore(nil, nil))
	}
	dr.Fields = fs

	return nil
}

func (d *DataRecord) Clone() DataRecord {
	fs := make([]Field, 0)
	for _, f := range d.Fields {
		fs = append(fs, f.Clone())
	}

	return DataRecord{
		TemplateId: d.TemplateId,
		FieldCount: d.FieldCount,

		Fields: fs,
	}
}
