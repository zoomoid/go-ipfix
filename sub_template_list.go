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

func NewDefaultSubTemplateList() DataType {
	return newSubTemplateList(0)
}

// this constructor is purposefully not exported, as we want to force constructing entities to
// use the WithLength decorator on the DataType, as done in the field builder
func newSubTemplateList(length uint16) DataType {
	return &SubTemplateList{
		length:   length,
		semantic: SemanticUndefined,
	}
}

type SubTemplateList struct {
	isVariableLength bool

	semantic ListSemantic

	templateId uint16

	// length is the length of the records nested in the SubTemplateList in bytes
	// It does not include the semantic and templateId, which are statically added
	// when using SubTemplateList.Length()
	length uint16

	// value DataRecord
	value []DataRecord

	templateManager TemplateCache

	// observationDomainId is used for scoping templates in their manager
	// it is required for looking up the template belonging to this types templateId
	//
	// Note that observationDomainId is not included in the length calculation despite
	// being included in the metadata when encoding to JSON. This is due to the fact
	// that the *odid* is bound at creation of the parent template, not during decoding
	// as the payload of a data record does not carry any information about the observation
	// domain id. In particular, Subtemplates assume the odid of the parent template.
	observationDomainId uint32
}

func (t *SubTemplateList) String() string {
	if t.value == nil {
		return "nil"
	}
	drs := make([]string, 0)
	for _, dr := range t.value {
		drs = append(drs, dr.String())
	}

	return fmt.Sprintf("SubTemplateList(%d,%d,%s){%d}[%s]", t.templateId, t.Length(), t.semantic, t.observationDomainId, strings.Join(drs, " "))
}

func (t *SubTemplateList) Type() string {
	return "subTemplateList"
}

func (t *SubTemplateList) Value() interface{} {
	return t.value
}

func (t *SubTemplateList) SetValue(v any) DataType {
	// TODO(zoomoid): is this safe to assert? can we cleanly extract a slice of subTemplateListContent from a consolidated field?
	b, ok := v.([]DataRecord)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	l := uint16(0)
	for _, e := range b {
		for _, f := range e.Fields {
			l += f.Length()
		}
	}
	t.length = l
	return t
}

var (
	subTemplateListHeaderLength uint16 = 3
)

func (t *SubTemplateList) Length() uint16 {
	var length uint16
	for _, record := range t.value {
		length += record.Length()
	}
	return length + subTemplateListHeaderLength
}

func (*SubTemplateList) DefaultLength() uint16 {
	return subTemplateListHeaderLength
}

func (t *SubTemplateList) Clone() DataType {
	vs := make([]DataRecord, 0)
	for _, el := range t.value {
		vs = append(vs, el.Clone())
	}
	return &SubTemplateList{
		value:               vs,
		isVariableLength:    t.isVariableLength,
		semantic:            t.semantic,
		templateId:          t.templateId,
		length:              t.length,
		observationDomainId: t.observationDomainId,
		templateManager:     t.templateManager,
	}
}

func (t *SubTemplateList) WithLength(length uint16) DataTypeConstructor {
	return func() DataType {
		return newSubTemplateList(length)
	}
}

func (t *SubTemplateList) SetLength(length uint16) DataType {
	t.length = length
	return t
}

func (t *SubTemplateList) IsReducedLength() bool {
	return false
}

func (t *SubTemplateList) SetSemantic(semantic ListSemantic) *SubTemplateList {
	t.semantic = semantic
	return t
}

func (t *SubTemplateList) Semantic() ListSemantic {
	return t.semantic
}

func (t *SubTemplateList) TemplateID() uint16 {
	return t.templateId
}

func (t *SubTemplateList) Elements() []DataRecord {
	return t.value
}

func (t *SubTemplateList) Decode(r io.Reader) (err error) {
	// semantic and listBuffer are included in the length field preceeding
	// when using variable-length encoding
	err = binary.Read(r, binary.BigEndian, &t.semantic)
	if err != nil {
		return fmt.Errorf("failed to read list semantic in %T, %w", t, err)
	}

	err = binary.Read(r, binary.BigEndian, &t.templateId)
	if err != nil {
		return fmt.Errorf("failed to read template id in %T, %w", t, err)
	}

	if t.templateManager == nil {
		return fmt.Errorf("failed to get template (%d,%d), manager is nil", t.observationDomainId, t.templateId)
	}

	tmpl, err := t.templateManager.Get(context.TODO(), TemplateKey{
		ObservationDomainId: t.observationDomainId,
		TemplateId:          t.templateId,
	})
	if err != nil {
		return fmt.Errorf("failed to get template (%d,%d) from manager in %T, %w", t.observationDomainId, t.templateId, t, err)
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

	if t.length-subTemplateListHeaderLength <= 0 {
		// subTemplateList is empty, dont do anything else than setting
		// the value to an empty slice of data records
		// Reading from an empty (also zero-length) bytes.Buffer returns io.EOF,
		// which we catch explicitly with this
		t.value = records
		return nil
	}

	// now, as either the FixedLengthField or Field.Decode() in the case of variable-length
	// fields already determined the length of this DataType, use this length parameter to
	// read data.
	lb := make([]byte, t.length-subTemplateListHeaderLength) // we already read 3 bytes from the buffer of valid data for the stl
	_, err = r.Read(lb)
	if err != nil {
		return fmt.Errorf("failed to read from field buffer for decoding %T, %w", t, err)
	}
	listBuffer := bytes.NewBuffer(lb)
	for listBuffer.Len() > 0 {
		dataFields, err := DecodeUsingTemplate(listBuffer, fields)
		if err != nil {
			return fmt.Errorf("failed to decode sub template from list buffer in %T, %w", t, err)
		}

		subDataRecord := DataRecord{
			Fields: dataFields,
		}

		records = append(records, subDataRecord)
	}

	t.value = records
	return nil
}

func (t *SubTemplateList) Encode(w io.Writer) (n int, err error) {
	// header
	b := make([]byte, 0)
	b = append(b, byte(t.semantic))

	b = binary.BigEndian.AppendUint16(b, t.templateId)

	n, err = w.Write(b)
	if err != nil {
		return
	}

	for _, r := range t.Elements() {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func (t *SubTemplateList) NewBuilder() TemplateListTypeBuilder {
	return &subTemplateListBuilder{}
}

type subTemplateListMetadata struct {
	Semantic            ListSemantic `json:"semantic" yaml:"semantic"`
	TemplateId          uint16       `json:"template_id" yaml:"templateId"`
	ObservationDomainId uint32       `json:"observation_domain_id" yaml:"observationDomainId"`
}

type marshalledSubTemplateList struct {
	Metadata subTemplateListMetadata `json:"metadata" yaml:"metadata"`
	Records  []DataRecord            `json:"records" yaml:"records"`
}

func (t *SubTemplateList) MarshalJSON() ([]byte, error) {
	return json.Marshal(marshalledSubTemplateList{
		Metadata: subTemplateListMetadata{
			Semantic:   t.semantic,
			TemplateId: t.templateId,
		},
		Records: t.value,
	})
}

func (t *SubTemplateList) UnmarshalJSON(in []byte) error {
	tt := marshalledSubTemplateList{}
	err := json.Unmarshal(in, &tt)
	if err != nil {
		return err
	}
	t.value = tt.Records
	l := uint16(0)
	for _, e := range t.value {
		for _, f := range e.Fields {
			l += f.Length()
		}
	}
	t.length = l
	t.templateId = tt.Metadata.TemplateId
	t.semantic = tt.Metadata.Semantic
	t.observationDomainId = tt.Metadata.ObservationDomainId
	// cannot restore functional fields such as TemplateCache and FieldCache from JSON
	return nil
}

type subTemplateListBuilder struct {
	templateManager TemplateCache
	fieldManager    FieldCache

	observationDomainId uint32
}

func (t *subTemplateListBuilder) WithTemplateManager(templateManager TemplateCache) TemplateListTypeBuilder {
	t.templateManager = templateManager
	return t
}

func (t *subTemplateListBuilder) WithFieldManager(fieldManager FieldCache) TemplateListTypeBuilder {
	t.fieldManager = fieldManager
	return t
}

func (t *subTemplateListBuilder) WithObservationDomain(id uint32) TemplateListTypeBuilder {
	t.observationDomainId = id
	return t
}

func (t *subTemplateListBuilder) Complete() DataTypeConstructor {
	return func() DataType {
		return &SubTemplateList{
			templateManager:     t.templateManager,
			observationDomainId: t.observationDomainId,
			semantic:            SemanticUndefined,
		}
	}
}

var _ TemplateListTypeBuilder = &subTemplateListBuilder{}
var _ TemplateListType = &SubTemplateList{}
var _ DataTypeConstructor = NewDefaultSubTemplateList
