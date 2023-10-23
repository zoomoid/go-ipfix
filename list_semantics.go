package ipfix

// ListSemantic is the type capturing the IANA-assigned list semantics as defined by RFC 6313
type ListSemantic uint8

const (
	// The "noneOf" structured data type semantic specifies that none of the
	// elements are actual properties of the Data Record.
	//
	// For example, a mediator might want to report to a Collector that a
	// specific Flow is suspicious, but that it checked already that this
	// Flow does not belong to the attack type 1, attack type 2, or attack
	// type 3.  So this Flow might need some further inspection.  In such a
	// case, the mediator would report the Flow Record with a basicList
	// composed of (attack type 1, attack type 2, attack type 3) and the
	// respective structured data type semantic of "noneOf".
	SemanticNoneOf ListSemantic = 0

	// The "exactlyOneOf" structured data type semantic specifies that only
	// a single element from the structured data is an actual property of
	// the Data Record.  This is equivalent to a logical XOR operation.
	SemanticExactlyOneOf ListSemantic = 1

	// The "oneOrMoreOf" structured data type semantic specifies that one or
	// more elements from the list in the structured data are actual
	// properties of the Data Record.  This is equivalent to a logical OR
	// operation.
	SemanticOneOrMoreOf ListSemantic = 2

	// The "allOf" structured data type semantic specifies that all of the
	// list elements from the structured data are actual properties of the
	// Data Record.
	//
	// For example, if a Record contains a basicList of outgoing interfaces
	// with the "allOf" semantic, then the observed Flow is typically a
	// multicast Flow where each packet in the Flow has been replicated to
	// each outgoing interface in the basicList.
	SemanticAllOf ListSemantic = 3

	// The "ordered" structured data type semantic specifies that elements
	// from the list in the structured data are ordered.
	//
	// For example, an Exporter might want to export the AS10 AS20 AS30 AS40
	// BGP AS-PATH.  In such a case, the Exporter would report a basicList
	// composed of (AS10, AS20, AS30, AS40) and the respective structured
	// data type semantic of "ordered".
	SemanticOrdered ListSemantic = 4

	// The "undefined" structured data type semantic specifies that the
	// semantic of list elements is not specified and that, if a semantic
	// exists, then it is up to the Collecting Process to draw its own
	// conclusions.  The "undefined" structured data type semantic, which is
	// the default value, is used when no other structured data type
	// semantic applies.

	// For example, a mediator that wants to translate IPFIX [RFC5101] into
	// the export of structured data according to the specifications in this
	// document doesn't know what the semantic is; it can only guess, as the
	// IPFIX specifications [RFC5101] does not contain any semantic.
	// Therefore, the mediator should use the "undefined" semantic.
	SemanticUndefined ListSemantic = 255
)

func (s ListSemantic) String() string {
	switch s {
	case SemanticNoneOf:
		return "noneOf"
	case SemanticExactlyOneOf:
		return "exactlyOneOf"
	case SemanticOneOrMoreOf:
		return "oneOrMoreOf"
	case SemanticAllOf:
		return "allOf"
	case SemanticOrdered:
		return "ordered"
	case SemanticUndefined:
		return "undefined"
	default:
		return "unassigned"
	}
}

// MarshalText implements encoding.Marshaler to convert a ListSemantic
// instance into a string representation
func (s ListSemantic) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.Unmarshaler to convert the string
// representation of a ListSemantic into its proper type
func (s *ListSemantic) UnmarshalText(in []byte) error {
	st := string(in)
	switch st {
	case "noneOf":
		*s = SemanticNoneOf
	case "exactlyOneOf":
		*s = SemanticExactlyOneOf
	case "oneOrMoreOf":
		*s = SemanticOneOrMoreOf
	case "allOf":
		*s = SemanticAllOf
	case "ordered":
		*s = SemanticOrdered
	case "undefined":
		*s = SemanticUndefined
	default:
	}
	return nil
}
