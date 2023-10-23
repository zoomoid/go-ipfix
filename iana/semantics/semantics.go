package semantics

import (
	"encoding"
	"fmt"
)

type Semantic int

// TODO(zoomoid): this has ambiguous behaviour for the case of explicit "Undefined"
// and implicit undefined (i.e. not part of the switch), where "" and "unassigned" are
// two different literals
const (
	Undefined Semantic = iota

	Default

	Quantity

	// TODO(zoomoid): we want to add labels for fields that are declared as either totalCounters or deltaCounters
	// such that the ingestor can take all those fields and insert them into InfluxDB as those are properly
	// "measureable"
	TotalCounter
	DeltaCounter

	Identifier

	Flags

	List

	SNMPCounter

	SNMPGauge
)

var supportedSemantics []Semantic = []Semantic{
	Undefined,
	Default,
	Quantity,
	TotalCounter,
	DeltaCounter,
	Identifier,
	Flags,
	List,
	SNMPCounter,
	SNMPGauge,
}

func SupportedSemantics() []Semantic {
	return supportedSemantics
}

// TODO(zoomoid): this has ambiguous behaviour for the case of explicit "Undefined"
// and implicit undefined (i.e. not part of the switch), where "" and "unassigned" are
// two different literals
func (s Semantic) String() string {
	switch s {
	case Undefined:
		return ""
	case Default:
		return "default"
	case Quantity:
		return "quantity"
	case TotalCounter:
		return "totalCounter"
	case DeltaCounter:
		return "deltaCounter"
	case Identifier:
		return "identifier"
	case Flags:
		return "flags"
	case List:
		return "list"
	case SNMPCounter:
		return "snmpCounter"
	case SNMPGauge:
		return "snmpGauge"
	default:
		return "unassigned"
	}
}

func FromNumber(i uint8) Semantic {
	switch i {
	case 0:
		return Default
	case 1:
		return Quantity
	case 2:
		return TotalCounter
	case 3:
		return DeltaCounter
	case 4:
		return Identifier
	case 5:
		return Flags
	case 6:
		return List
	case 7:
		return SNMPCounter
	case 8:
		return SNMPGauge
	default:
		return Undefined
	}
}

var _ fmt.Stringer = Semantic(0)
var _ encoding.TextMarshaler = Semantic(0)

// var _ encoding.TextUnmarshaler = Semantic(0)

func (s Semantic) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Semantic) UnmarshalText(in []byte) error {
	*s = Parse(string(in))
	return nil
}

func Parse(semantic string) Semantic {
	switch semantic {
	case "default":
		return Default
	case "quantity":
		return Quantity
	case "totalCounter":
		return TotalCounter
	case "deltaCounter":
		return DeltaCounter
	case "identifier":
		return Identifier
	case "flags":
		return Flags
	case "list":
		return List
	case "snmpCounter":
		return SNMPCounter
	case "snmpGauge":
		return SNMPGauge
	default:
		return Undefined
	}
}
