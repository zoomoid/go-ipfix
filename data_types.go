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
	"errors"
	"fmt"
	"io"
)

var (
	ErrUndefinedEncoding = errors.New("undefined data type encoding")
)

type DataType interface {
	json.Marshaler
	json.Unmarshaler
	fmt.Stringer

	// Type returns a string representation of the DataTypes type.
	// The name is used in Recover() to reconstruct a Field's DataType constructor
	Type() string

	// Length returns the actual length of the value captured by the DataType. This
	// is important because reduced-length and variable-length-encoded fields
	// will have different lengths than the spec prescribes
	Length() uint16

	// DefaultLength returns the DataType's length as defined by the specification
	DefaultLength() uint16

	// Decode reads a fixed number of bytes from the reader and decodes the value
	// data-type-dependant. Returns the first error that occurs during decoding.
	Decode(io.Reader) (int, error)

	// Encode writes a data type in IPFIX binary format to a given writer.
	// It returns the number of written bytes, and an error if an error occurred
	Encode(io.Writer) (int, error)

	// Value returns the internal value of the DataType. Marshalling the DataType will
	// only marshal the internal value to a marshallable type. Additional information
	// such as reduced-length encoding defined on the field and references to FieldManager
	// and TemplateManager are NOT marshalled.
	Value() interface{}

	// IsReducedLength indicates that a field has been constructed with
	// a custom, reduced-length encoding as per RFC 7011.
	// Therefore, when restoring a Field from a ConsolidatedField, if true,
	// Length() can be used to reconstruct the custom length parameter with a
	// FieldBuilder.
	IsReducedLength() bool

	// WithLength decorates the DataTypeConstructor with a defined length such that every
	// DataType created with the constructor has a predefined length.
	//
	// NOTE that WithLength does not copy any other internal properties of a DataType, e.g.,
	// already defined list semantics for basicList, subTemplateList, or subTemplateMultiList.
	// The only (non-testing) usage of WithLength is currently in FieldBuilder.Complete(), which
	// creates fresh Fields anyways so any already decoded or otherwise initialized properties
	// do not apply here. All other references should probably use the provided setters instead
	// of decorators because of this non-copy nature of the function.
	WithLength(uint16) DataTypeConstructor

	SetLength(uint16) DataType

	// Clone copies all copyable values on the DataType into a fresh one for creating new instances
	// of DataTypes
	Clone() DataType

	// SetValue sets the internal value on the DataType. SetValue panics if v cannot be asserted to the
	// internal type (this type safety should be ensured in the conversion from ConsolidatedField to Field)
	SetValue(v any) DataType
}

// LookupConstructor is an accessor to the private internal, but global map of currently known
// IPFIX abstract data types.
//
// If no constructor is associated with the given name, LookupConstructor panics. This behavior
// is to be discussed and potentially amended.
func LookupConstructor(name string) DataTypeConstructor {
	c, ok := constructors[name]
	if !ok {
		panic(fmt.Errorf("data type constructor not defined: %s", name))
	}
	return c
}

// SupportedTypes returns a slice containing all currently known DataType constructors.
func SupportedTypes() []DataTypeConstructor {
	cs := make([]DataTypeConstructor, len(constructors))
	idx := 0
	for _, c := range constructors {
		cs[idx] = c
	}
	return cs
}

// DataTypeConstructor is a type capturing the 0-adic constructor function for a new DataType.
// All DataTypes should, aside from their implementation of DataType's methods, also provide
// such a constructor function for unified instantiation.
//
// Mechanisms of dependency injection can also lead DataTypes to implement decorators that return
// new DataTypeConstructor functions with parameters curried inside the constructor function's closure.
type DataTypeConstructor func() DataType

// DataTypeFromNumber looks up the default constructor for each of the currently known
// IPFIX abstract data types (both in RFC 7011 and RFC 6313) by their IANA-assigned
// identifier.
// If an id is given that is NOT in the lookup table, DataTypeFromNumber panics.
// This behaviour is due to no better error handling mechanism currently existing
// in the call path of this function.
//
// TODO(zoomoid): rethink if panicking is the best idea here.
func DataTypeFromNumber(id uint8) DataTypeConstructor {
	switch id {
	case 0:
		return NewOctetArray
	case 1:
		return NewUnsigned8
	case 2:
		return NewUnsigned16
	case 3:
		return NewUnsigned32
	case 4:
		return NewUnsigned64
	case 5:
		return NewSigned8
	case 6:
		return NewSigned16
	case 7:
		return NewSigned32
	case 8:
		return NewSigned64
	case 9:
		return NewFloat32
	case 10:
		return NewFloat64
	case 11:
		return NewBoolean
	case 12:
		return NewMacAddress
	case 13:
		return NewString
	case 14:
		return NewDateTimeSeconds
	case 15:
		return NewDateTimeMilliseconds
	case 16:
		return NewDateTimeMicroseconds
	case 17:
		return NewDateTimeNanoseconds
	case 18:
		return NewIPv4Address
	case 19:
		return NewIPv6Address
	case 20:
		return NewBasicList
	case 21:
		return NewDefaultSubTemplateList
	case 22:
		return NewDefaultSubTemplateMultiList
	default:
		err := fmt.Errorf("DataType ID %d is not assigned", id)
		// logger.V(1).Error(err, "cannot use id for retrieving data type", "id", id)
		// panic from here because we have no proper error handling propagation from here
		// a controller configured to recover from panics will pick this up.
		panic(err)
	}
}

var constructors map[string]DataTypeConstructor = map[string]DataTypeConstructor{
	"octetArray":           NewOctetArray,
	"unsigned8":            NewUnsigned8,
	"unsigned16":           NewUnsigned16,
	"unsigned32":           NewUnsigned32,
	"unsigned64":           NewUnsigned64,
	"signed8":              NewSigned8,
	"signed16":             NewSigned16,
	"signed32":             NewSigned32,
	"signed64":             NewSigned64,
	"float32":              NewFloat32,
	"float64":              NewFloat64,
	"boolean":              NewBoolean,
	"macAddress":           NewMacAddress,
	"string":               NewString,
	"dateTimeSeconds":      NewDateTimeSeconds,
	"dateTimeMilliseconds": NewDateTimeMilliseconds,
	"dateTimeMicroseconds": NewDateTimeMicroseconds,
	"dateTimeNanoseconds":  NewDateTimeNanoseconds,
	"ipv4Address":          NewIPv4Address,
	"ipv6Address":          NewIPv6Address,
	"basicList":            NewBasicList,
	"subTemplateList":      NewDefaultSubTemplateList,
	"subTemplateMultiList": NewDefaultSubTemplateMultiList,
}

var _ json.Marshaler = DataType(nil)
var _ json.Unmarshaler = DataType(nil)
var _ fmt.Stringer = DataType(nil)
