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
	"io"
	"strings"

	"github.com/zoomoid/go-ipfix/iana/semantics"
)

type BidirectionalField interface {
	// IsReversible returns true if the field's underlying information element is
	// *not* contained in the list of irreversible information elements as per RFC
	// 5103.
	//
	// Note that this function is only practical for information elements described
	// in RFC 5103, i.e., IEs assigned by IANA, because only for those, the semantics
	// of reversal are well-defined by the RFC. Enterprise-specific IEs may choose
	// to implement their own reversal semantics, as it is the case with e.g. CERT IEs
	Reversible() bool

	// Reversed returns the field's state with regards to RFC 5103, i.e., if the
	// field is used to carry biflow information for the reverse direction. Note that
	// this is also used for returning the field's name, indicating that a field is
	// reversed by prepending "reversed" in front of the name.
	Reversed() bool
}

// Field defines the interface for both types of IPFIX fields, either fixed-length or variable-length
// fields. They share most of their logic, except for encoding and decoding, for which the Encode
// and Decode methods transparently handle their underlying nature.
// Fixed-length fields are intuitively simpler, as their length and data type is defined by templates.
// Variable-length fields encode their data length in the first 1 or 3 bytes (short and long format).
//
// The interface also declares methods for converting a fixed-length field to a variable-length field,
// using Lift(). Though this is practically never done in plain IPFIX, it is convenient to have such
// a function in user space for conversion between both underlying field types. Note that the reverse
// direction, converting a variable-length field to a fixed-length field is NOT possible.
type Field interface {
	// Id returns the field id as defined
	Id() uint16

	// Name returns the name of the field
	Name() string

	// Value returns the underlying data type
	Value() DataType

	// SetValue sets the value on the internal DataType stored in the field
	SetValue(v any) Field

	// Type returns a string representation of the underlying DataType
	Type() string

	// PEN returns the private enterprise number of the field, if set, as a uint32 pointer,
	// and otherwise nil
	PEN() uint32

	// Constructor returns the data type constructor defined for the field
	Constructor() DataTypeConstructor

	// Length returns the semantically-aware length of the field
	Length() uint16

	// ObservationDomainId returns the ID bound to the field from the builder For
	// fields whose underlying data types are reliant on this ID, i.e.,
	// SubTemplateList and SubTemplateMultiList, this is required or otherwise
	// decoding will not work correctly.
	ObservationDomainId() uint32

	// Prototype returns a copy of the field's IE specification, i.e., the
	// prototype of the field. This can be used for cloning and copying of the
	// field while preserving semantics
	Prototype() InformationElement

	// Lift converts a field to a variable-length field, indicating this to the
	// data type constructor
	Lift() *VariableLengthField

	// Decode creates a DataType from the supplied constructor and decodes the
	// value from the Reader
	Decode(io.Reader) (int, error)

	// Encode writes a field in IPFIX binary format to a given writer.
	// It returns the number of written bytes, and an error if an error occurred
	Encode(io.Writer) (int, error)

	// Clone clones a field entirely by-value such that changing a Field in a data
	// record which is also used in a template record does not cause side effects.
	// The only thing copied by- reference are FieldManager and TemplateManager
	// instances.
	Clone() Field

	// SetScoped is the setter to be used when the field is a scope field.
	SetScoped() Field

	// IsScope returns true if the field is set to be a scope field. This is
	// useful when encoding to a map structure in e.g. encoding/libfds, as scope
	// fields with the same name would be lost due to map key collisions.
	// Therefore we use this boolean to compute a key that is resistent to the
	// collision.
	IsScope() bool

	BidirectionalField

	// Consolidate converts the field into a value easily serialized, i.e., by
	// removing functions and encoding whether the field is a fixed length or
	// variable length variant.
	Consolidate() ConsolidatedField

	json.Marshaler
	json.Unmarshaler
}

type ConsolidatedField struct {
	Id   uint16 `json:"id"`
	Name string `json:"name,omitempty"`

	PEN uint32 `json:"pen"`

	// Length contains the DataType's length in bytes. Notably, if the field is
	// encoded with reduced length, this field captures the necessary information
	// to reconstruct the field later on.
	Length uint16 `json:"length"`

	// To reconstruct the original field type from a consolidated one
	IsVariableLength bool `json:"is_variable_length,omitempty"`

	ObservationDomainId uint32 `json:"observation_domain_id,omitempty"`

	// Value interface{} `json:"value,omitempty"`
	Value *json.RawMessage `json:"value,omitempty"`

	// Type is a serialized form of the DataType underlying the field. Note that
	// this string representation is used for Restore(), however, additional
	// information such as reduced-length encoding requires more information to be
	// embedded
	Type string `json:"type,omitempty"`

	IsScope bool `json:"is_scope,omitempty"`
}

// var _ json.Unmarshaler = &ConsolidatedField{}
// var _ json.Marshaler = &ConsolidatedField{}

// func (cf *ConsolidatedField) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(cf)
// }

// func (cf *ConsolidatedField) UnmarshalJSON(in []byte) error {
// 	return json.Unmarshal(in, cf)
// }

var dataTypesWithListSemantics map[string]struct{} = map[string]struct{}{
	(&BasicList{}).Type():            {},
	(&SubTemplateList{}).Type():      {},
	(&SubTemplateMultiList{}).Type(): {},
}

// Restore creates a Field from a ConsolidatedField again, by deciding whether to use an
// underlying variable length or fixed length struct.
// Restore also recreates the constructor function from the type string left on the
// Consolidated field, as well as restoring the internal value of a DataType
func (cf *ConsolidatedField) Restore(fieldManager FieldCache, templateManager TemplateCache) Field {
	constr := LookupConstructor(cf.Type)

	// construct an ad-hoc information element. We don't assume it belongs to any specific registry, that's
	// why we omit lookups here
	ie := InformationElement{
		Constructor: constr,
	}

	ie.Name = cf.Name

	var reverse bool
	// Consolidating a reverse field sets the PEN to the PEN reserved for reverse fields
	if strings.HasPrefix(cf.Name, "reverse") && cf.PEN == ReversePEN {
		// reset PEN to the default IANA namespace, but preserve the information
		// that the field is reversed in a separate variable
		reverse = true
		cf.PEN = 0
	}

	ie.Id = cf.Id

	// if DataType type is inherently a list type...
	if _, isListSemantic := dataTypesWithListSemantics[cf.Type]; isListSemantic {
		ie.Semantics = semantics.List
	}

	builder := NewFieldBuilder(ie).
		SetLength(cf.Length).
		SetObservationDomain(cf.ObservationDomainId).
		SetPEN(cf.PEN).
		SetReversed(reverse).
		SetFieldManager(fieldManager).
		SetTemplateManager(templateManager)

	f := builder.Complete()

	// TODO(zoomoid): this does not check the sanity of the values! currently,
	// when unmarshalling a basicList, this will not work because json.Unmarshal
	// in the Field.UnmarshalJSON unwraps the JSON to []interface{}, but this is
	// not assignable to the value field of BasicList, as it expects Fields.
	if v := cf.Value; v != nil {
		err := f.Value().UnmarshalJSON(*v)
		if err != nil {
			// TODO(zoomoid): panic behaviour of SetValue
			panic(err)
		}
	}

	return f
}
