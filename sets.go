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
	"errors"
	"fmt"
	"io"
)

type Set struct {
	SetHeader `json:",inline" yaml:",inline"`
	Kind      string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Set set `json:"flow_set,omitempty"`
}

// The Kind* constants are used for unmarshalling of JSON records to denote the specific type
// into which the elements of a set should be unmarshalled in.
const (
	KindDataSet            string = "DataSet"
	KindTemplateSet        string = "TemplateSet"
	KindOptionsTemplateSet string = "OptionsTemplateSet"
)

var _ fmt.Stringer = &Set{}
var _ json.Marshaler = &Set{}
var _ json.Unmarshaler = &Set{}

func (s *Set) String() string {
	return fmt.Sprintf("%s<ID=%d,Records=%d>%s", s.Kind, s.Id, s.Set.Length(), s.Set)
}

func (s *Set) MarshalJSON() ([]byte, error) {
	type ifs struct {
		Id uint16 `json:"id,omitempty"`

		Length uint16 `json:"length,omitempty"`

		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

		Records json.RawMessage `json:"records,omitempty" yaml:"records,omitempty"`
	}

	t := &ifs{
		Id:     s.Id,
		Length: s.Length,
		Kind:   s.Kind,
	}

	var set []byte
	var err error
	switch ff := s.Set.(type) {
	case *DataSet:
		set, err = json.Marshal(ff.Records)
	case *TemplateSet:
		set, err = json.Marshal(ff.Records)
	case *OptionsTemplateSet:
		set, err = json.Marshal(ff.Records)
	}
	if err != nil {
		return nil, err
	}

	t.Records = json.RawMessage(set)

	return json.Marshal(t)
}

func (s *Set) Encode(w io.Writer) (n int, err error) {
	// header
	l := make([]byte, 2)
	binary.BigEndian.PutUint16(l, s.SetHeader.Id)
	ln, err := w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	l = make([]byte, 2)
	binary.BigEndian.PutUint16(l, s.SetHeader.Length)
	ln, err = w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	// body
	if s.Set != nil {
		bn, err := s.Set.Encode(w)
		n += bn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (s *Set) UnmarshalJSON(in []byte) error {
	type ifs struct {
		SetHeader `json:",inline" yaml:",inline"`
		Kind      string `json:"kind,omitempty" yaml:"kind,omitempty"`

		Records json.RawMessage `json:"records,omitempty" yaml:"records,omitempty"`
	}

	t := &ifs{}
	err := json.Unmarshal(in, t)
	if err != nil {
		return err
	}

	var ff set
	switch t.Kind {
	case KindDataSet:
		dfs := &DataSet{}
		err = json.Unmarshal(t.Records, &dfs.Records)
		if err != nil {
			break
		}
		ff = dfs
	case KindTemplateSet:
		tfs := &TemplateSet{}
		err = json.Unmarshal(t.Records, &tfs.Records)
		if err != nil {
			break
		}
		ff = tfs
	case KindOptionsTemplateSet:
		iotfs := &OptionsTemplateSet{}
		err = json.Unmarshal(t.Records, &iotfs.Records)
		if err != nil {
			break
		}
		ff = iotfs
	}
	if err != nil {
		return fmt.Errorf("failed to unmarshal into Records, %w", err)
	}

	*s = Set{
		SetHeader: t.SetHeader,
		Kind:      t.Kind,
		Set:       ff,
	}
	return nil
}

type DataSet struct {
	Records []DataRecord `json:"records,omitempty" yaml:"records,omitempty"`

	fieldCache    FieldCache
	templateCache TemplateCache

	template *Template
}

func (d *DataSet) String() string {
	sl := make([]string, 0, len(d.Records))
	for _, dr := range d.Records {
		sl = append(sl, dr.String())
	}

	return fmt.Sprintf("%v", sl)
}

func (d *DataSet) Length() int {
	return len(d.Records)
}

func (d *DataSet) Encode(w io.Writer) (n int, err error) {
	for _, r := range d.Records {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (d *DataSet) With(t *Template) *DataSet {
	d.template = t
	return d
}

func (d *DataSet) Decode(r io.Reader) (n int, err error) {
	if d.template == nil {
		return 0, errors.New("no template bound to data record")
	}

	for {
		dr := DataRecord{
			template:   d.template,
			TemplateId: d.template.TemplateId,
		}
		m, err := dr.Decode(r)
		n += m
		if err != nil && err == io.EOF {
			return n, err
		}
		d.Records = append(d.Records, dr)
		if err == io.EOF {
			break
		}
	}

	return
}

type TemplateSet struct {
	Records []TemplateRecord `json:"records,omitempty" yaml:"records,omitempty"`

	fieldCache    FieldCache
	templateCache TemplateCache
}

func (d *TemplateSet) String() string {
	sl := make([]string, 0, len(d.Records))
	for _, tr := range d.Records {
		sl = append(sl, tr.String())
	}
	return fmt.Sprintf("%v", sl)
}

func (d *TemplateSet) Length() int {
	return len(d.Records)
}

func (d *TemplateSet) Encode(w io.Writer) (n int, err error) {
	for _, r := range d.Records {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (d *TemplateSet) Decode(r io.Reader) (n int, err error) {
	d.Records = make([]TemplateRecord, 0)
	// "as long as there's set header data (Set ID, Length)"
	for {
		templateRecord := TemplateRecord{}

		m, err := templateRecord.Decode(r)
		n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, err
		}
	}
	return
}

type OptionsTemplateSet struct {
	Records []OptionsTemplateRecord `json:"records,omitempty" yaml:"records,omitempty"`

	fieldCache    FieldCache
	templateCache TemplateCache
}

func (d *OptionsTemplateSet) String() string {
	ss := make([]string, 0, len(d.Records))
	for _, otr := range d.Records {
		ss = append(ss, otr.String())
	}

	return fmt.Sprintf("%v", ss)
}

func (d *OptionsTemplateSet) Length() int {
	return len(d.Records)
}

func (d *OptionsTemplateSet) Encode(w io.Writer) (n int, err error) {
	for _, r := range d.Records {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (d *OptionsTemplateSet) Decode(r io.Reader) (n int, err error) {
	d.Records = make([]OptionsTemplateRecord, 0)
	// TODO(zoomoid): maybe we need this for bound checks...
	// for r.Len() >= 4 {
	for {
		record := OptionsTemplateRecord{}

		m, err := record.Decode(r)
		n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, err
		}
	}
	return
}

type set interface {
	fmt.Stringer

	Length() int

	Encode(io.Writer) (int, error)
	// Decode(io.Reader) (int, error)
}
