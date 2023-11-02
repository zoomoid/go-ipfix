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

	"github.com/zoomoid/go-ipfix/iana/semantics"
)

const (
	// VariableLength is the constant used for denoting a field being variable-length encoded
	// in template records before the length is known.
	VariableLength uint16 = 0xFFFF
)

type FieldBuilder struct {
	prototype *InformationElement
	length    uint16

	reverse bool

	observationDomainId uint32

	fieldManager    FieldCache
	templateManager TemplateCache
}

var _ json.Marshaler = &FieldBuilder{}
var _ json.Unmarshaler = &FieldBuilder{}

func NewFieldBuilder(ie *InformationElement) *FieldBuilder {
	return &FieldBuilder{
		prototype: ie,
	}
}

func (b *FieldBuilder) GetIE() *InformationElement {
	return b.prototype
}

func (b *FieldBuilder) SetObservationDomain(id uint32) *FieldBuilder {
	b.observationDomainId = id
	return b
}

func (b *FieldBuilder) SetFieldManager(fieldManager FieldCache) *FieldBuilder {
	b.fieldManager = fieldManager
	return b
}

func (b *FieldBuilder) SetTemplateManager(templateManager TemplateCache) *FieldBuilder {
	b.templateManager = templateManager
	return b
}

// SetLength sets the field's length. This handles 0xFF as variable
func (b *FieldBuilder) SetLength(length uint16) *FieldBuilder {
	b.length = length
	return b
}

// SetPEN sets the field's Private Enterprise Number
func (b *FieldBuilder) SetPEN(pen uint32) *FieldBuilder {
	b.prototype.EnterpriseId = pen
	return b
}

func (b *FieldBuilder) SetReversed(isReverse bool) *FieldBuilder {
	b.reverse = isReverse
	return b
}

func (b *FieldBuilder) Complete() Field {
	constructorBuilder := NewDataTypeBuilder(b.prototype.Constructor).SetLength(b.length)
	// if the semantic of the field is a List, then decorate their constructors with
	if b.prototype.Semantics == semantics.List {
		constructorBuilder.
			SetObservationDomain(b.observationDomainId).
			SetFieldCache(b.fieldManager).
			SetTemplateCache(b.templateManager)
	}

	decoratedConstructor := constructorBuilder.Complete()

	if b.length == VariableLength {
		return &VariableLengthField{
			id:                  b.prototype.Id,
			name:                b.prototype.Name,
			constructor:         decoratedConstructor,
			observationDomainId: b.observationDomainId,
			pen:                 b.prototype.EnterpriseId,
			reversed:            b.reverse,
			fieldManager:        b.fieldManager,
			templateManager:     b.templateManager,
			prototype:           b.prototype,
		}
	} else {
		return &FixedLengthField{
			id:                  b.prototype.Id,
			name:                b.prototype.Name,
			constructor:         decoratedConstructor,
			reversed:            b.reverse,
			observationDomainId: b.observationDomainId,
			pen:                 b.prototype.EnterpriseId,
			fieldManager:        b.fieldManager,
			templateManager:     b.templateManager,
			prototype:           b.prototype,
		}
	}
}

type dataTypeBuilder struct {
	constructor DataTypeConstructor

	length uint16

	observationDomainId uint32

	fieldManager    FieldCache
	templateManager TemplateCache
}

func NewDataTypeBuilder(constructor DataTypeConstructor) *dataTypeBuilder {
	return &dataTypeBuilder{
		constructor: constructor,
	}
}

func (b *dataTypeBuilder) SetObservationDomain(id uint32) *dataTypeBuilder {
	b.observationDomainId = id
	return b
}

func (b *dataTypeBuilder) SetLength(length uint16) *dataTypeBuilder {
	b.length = length
	return b
}

func (b *dataTypeBuilder) SetFieldCache(fieldCache FieldCache) *dataTypeBuilder {
	b.fieldManager = fieldCache
	return b
}

func (b *dataTypeBuilder) SetTemplateCache(templateCache TemplateCache) *dataTypeBuilder {
	b.templateManager = templateCache
	return b
}

func (b *dataTypeBuilder) Complete() DataTypeConstructor {
	decoratedConstructor := b.constructor().WithLength(b.length)

	// ListType and TemplateListTypes are decorated additionally with FieldCache or TemplateCache
	switch lc := decoratedConstructor().(type) {
	case listType:
		decoratedConstructor = lc.
			NewBuilder().
			WithFieldCache(b.fieldManager).
			Complete()
	case templateListType:
		decoratedConstructor = lc.
			NewBuilder().
			WithFieldCache(b.fieldManager).
			WithTemplateCache(b.templateManager).
			WithObservationDomain(b.observationDomainId).
			Complete()
	}

	return decoratedConstructor
}

type consolidatedFieldBuilder struct {
	Prototype           *InformationElement `json:"prototype,omitempty"`
	ObservationDomainId uint32              `json:"observation_domain_id,omitempty"`
	Length              uint16              `json:"length,omitempty"`
	Reverse             bool                `json:"reverse,omitempty"`
}

func (b *FieldBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(consolidatedFieldBuilder{
		ObservationDomainId: b.observationDomainId,
		Length:              b.length,
		Prototype:           b.prototype,
		Reverse:             b.reverse,
	})
}

func (b *FieldBuilder) UnmarshalJSON(in []byte) error {
	s := &consolidatedFieldBuilder{}
	err := json.Unmarshal(in, s)
	if err != nil {
		return err
	}

	b.observationDomainId = s.ObservationDomainId
	b.length = s.Length
	b.prototype = s.Prototype
	b.reverse = s.Reverse

	return nil
}
