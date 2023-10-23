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
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func NewDefaultSubTemplateMultiList() DataType {
	return newSubTemplateMultiList(0)
}

func newSubTemplateMultiList(length uint16) DataType {
	return &SubTemplateMultiList{
		length:   length,
		semantic: SemanticUndefined,
	}
}

type SubTemplateMultiList struct {
	semantic ListSemantic

	length uint16

	value []subTemplateListContent

	templateManager TemplateCache

	// observationDomainId is used for scoping templates in their manager
	// it is required for looking up the template belonging to this types templateId
	observationDomainId uint32
}

func (t *SubTemplateMultiList) String() string {
	if t.value == nil {
		return "nil"
	}
	stl := make([]string, 0)
	for _, st := range t.value {
		stl = append(stl, st.String())
	}
	return fmt.Sprintf("SubTemplateMultiList(%d,%s){%d}[%s]", t.Length(), t.semantic, t.observationDomainId, strings.Join(stl, " "))
}

func (t *SubTemplateMultiList) Type() string {
	return "subTemplateMultiList"
}

func (t *SubTemplateMultiList) Value() interface{} {
	return t.value
}

func (t *SubTemplateMultiList) SetValue(v any) DataType {
	// TODO(zoomoid): is this safe to assert? can we cleanly extract a slice of subTemplateListContent from a consolidated field?
	b, ok := v.([]subTemplateListContent)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	l := uint16(0)
	for _, e := range b {
		for _, dr := range e.Values {
			// better computed length than before assuming e.Length being set
			l += dr.Length()
		}
	}
	t.length = l
	return t
}

func (t *SubTemplateMultiList) Length() uint16 {
	var length uint16
	for _, rr := range t.value {
		length += 4 // Template ID of this particular set of records + its length
		for _, r := range rr.Values {
			length += r.Length()
		}
	}
	return length + 1 // include 1 byte for semantics here
}

func (*SubTemplateMultiList) DefaultLength() uint16 {
	return 1
}

func (t *SubTemplateMultiList) Clone() DataType {
	vs := make([]subTemplateListContent, 0)
	for _, el := range t.value {
		vs = append(vs, el.Clone())
	}
	return &SubTemplateMultiList{
		semantic:            t.semantic,
		templateManager:     t.templateManager,
		observationDomainId: t.observationDomainId,
		length:              t.length,
		value:               vs,
	}
}

func (t *SubTemplateMultiList) WithLength(length uint16) DataTypeConstructor {
	return func() DataType {
		return newSubTemplateMultiList(length)
	}
}

// TODO(zoomoid): check if this is safely done with just the length of the elements,
// or if we need to include "headers" of each subtemplate as well
func (t *SubTemplateMultiList) SetLength(length uint16) DataType {
	t.length = length
	return t
}

func (*SubTemplateMultiList) IsReducedLength() bool {
	return false
}

func (t *SubTemplateMultiList) SetSemantic(semantic ListSemantic) *SubTemplateMultiList {
	t.semantic = semantic
	return t
}

func (t *SubTemplateMultiList) Semantic() ListSemantic {
	return t.semantic
}

func (t *SubTemplateMultiList) Elements() []subTemplateListContent {
	return t.value
}

func (t *SubTemplateMultiList) Decode(r io.Reader) error {
	var err error
	err = binary.Read(r, binary.BigEndian, &t.semantic)
	if err != nil {
		return fmt.Errorf("failed to read list semantic in %T, %w", t, err)
	}

	// exhaust the previously sliced buffer
	lb := make([]byte, t.length-1) // already read one byte of the list buffer for the semantic
	_, err = r.Read(lb)
	if err != nil {
		return fmt.Errorf("failed to read length in %T, %w", t, err)
	}
	listBuffer := bytes.NewBuffer(lb)

	for listBuffer.Len() > 0 {
		var subTemplateId, subTemplateLength uint16

		err = binary.Read(listBuffer, binary.BigEndian, &subTemplateId)
		if err != nil {
			return fmt.Errorf("failed to read sub template id in %T, %w", t, err)
		}

		if listBuffer.Len() == 0 {
			// no elements in subTemplateMultiList, abort...
			break
		}

		err = binary.Read(listBuffer, binary.BigEndian, &subTemplateLength)
		if err != nil {
			return fmt.Errorf("failed to read sub template length in %T, %w", t, err)
		}

		s := subTemplateListContent{
			TemplateId: subTemplateId,
			Length:     subTemplateLength,
		}

		if t.templateManager == nil {
			return fmt.Errorf("failed to get template (%d,%d), manager is nil", t.observationDomainId, subTemplateId)
		}

		tmpl, err := t.templateManager.Get(context.TODO(), TemplateKey{
			ObservationDomainId: t.observationDomainId,
			TemplateId:          subTemplateId,
		})
		if err != nil {
			return fmt.Errorf("failed to get template (%d,%d) from manager in %T, %w", t.observationDomainId, subTemplateId, t, err)
		}

		fields := make([]Field, 0)
		switch template := tmpl.Record.(type) {
		case *TemplateRecord:
			fields = append(fields, template.Fields...)
		case *OptionsTemplateRecord:
			fields = append(fields, template.Scopes...)
			fields = append(fields, template.Options...)
		default:
			return fmt.Errorf("expected either TemplateRecord or OptionsTemplateRecord, found %T", template)
		}

		records := make([]DataRecord, 0)
		for listBuffer.Len() > 0 {
			dataFields, err := DecodeUsingTemplate(listBuffer, fields)
			if err != nil {
				return err
			}
			subDataRecord := DataRecord{
				Fields: dataFields,
			}
			records = append(records, subDataRecord)
		}
		s.Values = records

		t.value = append(t.value, s)
	}
	return err
}

func (t *SubTemplateMultiList) Encode(w io.Writer) (n int, err error) {
	// header
	b := make([]byte, 0)
	b = append(b, byte(t.semantic))

	n, err = w.Write(b)
	if err != nil {
		return
	}

	for _, drs := range t.value {
		// subTemplateListContent element header
		l := make([]byte, 2)
		binary.BigEndian.PutUint16(l, drs.TemplateId)
		ln, err := w.Write(l)
		n += ln
		if err != nil {
			return n, err
		}
		l = make([]byte, 2)
		binary.BigEndian.PutUint16(l, drs.Length)
		ln, err = w.Write(l)
		n += ln
		if err != nil {
			return n, err
		}
		for _, r := range drs.Values {
			rn, err := r.Encode(w)
			n += rn
			if err != nil {
				return n, err
			}
		}
	}
	return n, err
}

func (t *SubTemplateMultiList) NewBuilder() TemplateListTypeBuilder {
	return &subTemplateMultiListBuilder{}
}

type subTemplateMultiListMetadata struct {
	Semantic            ListSemantic `json:"semantic" yaml:"semantic"`
	ObservationDomainId uint32       `json:"observation_domain_id" yaml:"observationDomainId"`
}

type marshalledSubTemplateMultiList struct {
	Metadata subTemplateMultiListMetadata `json:"metadata" yaml:"metadata"`
	Records  []subTemplateListContent     `json:"records,omitempty" yaml:"records"`
}

func (t *SubTemplateMultiList) MarshalJSON() ([]byte, error) {
	return json.Marshal(marshalledSubTemplateMultiList{
		Metadata: subTemplateMultiListMetadata{
			Semantic:            t.semantic,
			ObservationDomainId: t.observationDomainId,
		},
		Records: t.value,
	})
}

func (t *SubTemplateMultiList) UnmarshalJSON(in []byte) error {
	s := &marshalledSubTemplateMultiList{}
	err := json.Unmarshal(in, s)
	if err != nil {
		return err
	}
	t.value = s.Records
	l := uint16(0)
	for _, e := range t.value {
		for _, dr := range e.Values {
			// better computed length than before assuming e.Length being set
			l += dr.Length()
		}
	}
	t.length = l

	t.semantic = s.Metadata.Semantic
	t.observationDomainId = s.Metadata.ObservationDomainId
	return nil
}

type subTemplateListContent struct {
	TemplateId uint16       `json:"template_id" yaml:"templateId"`
	Length     uint16       `json:"length" yaml:"length"`
	Values     []DataRecord `json:"values" yaml:"values"`
}

var _ json.Marshaler = &subTemplateListContent{}
var _ json.Unmarshaler = &subTemplateListContent{}
var _ fmt.Stringer = &subTemplateListContent{}

func (s *subTemplateListContent) Len() int {
	return int(s.Length)
}

func (s *subTemplateListContent) Clone() subTemplateListContent {
	vs := make([]DataRecord, 0)
	for _, el := range s.Values {
		vs = append(vs, el.Clone())
	}

	return subTemplateListContent{
		TemplateId: s.TemplateId,
		Length:     s.Length,
		Values:     vs,
	}
}

func (s *subTemplateListContent) String() string {
	drs := make([]string, 0)
	for _, dr := range s.Values {
		drs = append(drs, dr.String())
	}
	return fmt.Sprintf("SubTemplate(%d/%d)[%s]", s.TemplateId, s.Len(), strings.Join(drs, " "))
}

func (s *subTemplateListContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *subTemplateListContent) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, s)
}

type subTemplateMultiListBuilder struct {
	templateManager TemplateCache
	fieldManager    FieldCache

	observationDomainId uint32
}

func (t *subTemplateMultiListBuilder) WithTemplateManager(templateManager TemplateCache) TemplateListTypeBuilder {
	t.templateManager = templateManager
	return t
}

func (t *subTemplateMultiListBuilder) WithFieldManager(fieldManager FieldCache) TemplateListTypeBuilder {
	t.fieldManager = fieldManager
	return t
}

func (t *subTemplateMultiListBuilder) WithObservationDomain(id uint32) TemplateListTypeBuilder {
	t.observationDomainId = id
	return t
}

func (t *subTemplateMultiListBuilder) Complete() DataTypeConstructor {
	return func() DataType {
		return &SubTemplateMultiList{
			templateManager:     t.templateManager,
			observationDomainId: t.observationDomainId,
			semantic:            SemanticUndefined,
		}
	}
}

var _ TemplateListTypeBuilder = &subTemplateMultiListBuilder{}
var _ TemplateListType = &SubTemplateMultiList{}
var _ DataTypeConstructor = NewDefaultSubTemplateMultiList
