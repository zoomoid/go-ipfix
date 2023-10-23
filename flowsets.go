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
)

type set interface {
	Length() int

	Encode(io.Writer) (int, error)
}

type Set struct {
	SetHeader `json:",inline" yaml:",inline"`
	Kind      string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Set set `json:"flow_set,omitempty"`
}

var _ json.Marshaler = &Set{}
var _ json.Unmarshaler = &Set{}

func (fs *Set) MarshalJSON() ([]byte, error) {
	type ifs struct {
		Id uint16 `json:"id,omitempty"`

		Length uint16 `json:"length,omitempty"`

		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

		Records json.RawMessage `json:"records,omitempty" yaml:"records,omitempty"`
	}

	t := &ifs{
		Id:     fs.Id,
		Length: fs.Length,
		Kind:   fs.Kind,
	}

	var set []byte
	var err error
	switch ff := fs.Set.(type) {
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

func (fs *Set) Encode(w io.Writer) (n int, err error) {
	// header
	l := make([]byte, 2)
	binary.BigEndian.PutUint16(l, fs.SetHeader.Id)
	ln, err := w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	l = make([]byte, 2)
	binary.BigEndian.PutUint16(l, fs.SetHeader.Length)
	ln, err = w.Write(l)
	n += ln
	if err != nil {
		return n, err
	}
	// body
	if fs.Set != nil {
		bn, err := fs.Set.Encode(w)
		n += bn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (fs *Set) UnmarshalJSON(in []byte) error {
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
	case KindDataRecord:
		dfs := &DataSet{}
		err = json.Unmarshal(t.Records, &dfs.Records)
		if err != nil {
			break
		}
		ff = dfs
	case KindTemplateRecord:
		tfs := &TemplateSet{}
		err = json.Unmarshal(t.Records, &tfs.Records)
		if err != nil {
			break
		}
		ff = tfs
	case KindOptionsTemplateRecord:
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

	*fs = Set{
		SetHeader: t.SetHeader,
		Kind:      t.Kind,
		Set:       ff,
	}
	return nil
}

type DataSet struct {
	Records []DataRecord `json:"records,omitempty" yaml:"records,omitempty"`
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

type TemplateSet struct {
	Records []TemplateRecord `json:"records,omitempty" yaml:"records,omitempty"`
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

type OptionsTemplateSet struct {
	Records []OptionsTemplateRecord `json:"records,omitempty" yaml:"records,omitempty"`
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

var (
	KindDataRecord            string = "DataRecord"
	KindTemplateRecord        string = "TemplateRecord"
	KindOptionsTemplateRecord string = "OptionsTemplateRecord"
)

var (
	KnownKinds map[string]struct{} = map[string]struct{}{
		"DataRecord":            {},
		"TemplateRecord":        {},
		"OptionsTemplateRecord": {},
	}
)
