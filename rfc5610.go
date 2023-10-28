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
	"fmt"

	"github.com/zoomoid/go-ipfix/iana/semantics"
	"github.com/zoomoid/go-ipfix/iana/units"
)

// definesNewInformationElement is used as a pre-check to determine if a DataRecord carries
// new a Information Element definition.
func definesInformationElement(dr DataRecord) bool {
	var idField Field
	var nameField Field
	for _, f := range dr.Fields {
		// IANA/informationElementId
		if f.Id() == 303 && f.PEN() == 0 {
			idField = f
			continue
		}
		// IANA/informationElementName
		if f.Id() == 341 && f.PEN() == 0 {
			nameField = f
			continue
		}
	}
	// This is only a heuristic! From what we observed in exporters providing support for RFC 5610,
	// these fields ought to be present. This reflects to our definition of IEs, where ID and Name
	// *should* be the only non-nil and non-empty fields *always*.
	return idField != nil && nameField != nil
}

// dataRecordToIE converts a data record containing (new) Information Elements to learn
// to IE objects to be added to field caches. This implements the learning mechanism of RFC 5610
// If the data record defines new information elements (RFC 5610), add them to FieldManager
// This is here because for yaf + (n)DPI, the exporter first sends a OptionsTemplate
// defining an options template for defining new Information Elements, i.e.,
//
//  1. privateEnterpriseNumber (0/346) [scope]
//  2. informationElementId (0/303) [scope]
//  3. informationElementDataType (0/339)
//  4. informationElementSemantics (0/344)
//  5. informationElementUnits (0/345)
//  6. informationElementRangeBegin (0/342)
//  7. informationElementRangeEnd (0/343)
//  8. informationElementName (0/341)
//  9. informationElementDescription (0/340)
//
// which is subsequently used to announce more IEs in the form of flow records
// carrying the information element definition. When we parse such a Data Record,
// we also want to "learn" the definition and add it to the field cache such that
// subsequently decoded fields carry semantics rather than "Unassigned" fields
func dataRecordToIE(dr DataRecord) (*InformationElement, error) {
	if !definesInformationElement(dr) {
		return nil, nil
	}

	ie := &InformationElement{}

	if f := dr.getFieldByName(0, "privateEnterpriseNumber"); f != nil {
		eid, ok := f.Value().(*Unsigned32)
		if !ok {
			return nil, fmt.Errorf("'privateEnterpriseId' field is not of type Unsigned32, cannot use field for deriving new IE")
		}
		// TODO(zoomoid): this uses the Value() function for cleanliness, but this might incur more steps in de-ref'ing the interface pointer
		ie.EnterpriseId = eid.Value().(uint32)
	} else {
		// TODO(zoomoid): this is not the end of the world, just assume that we are extending the IANA fields,
		// but to be sure, print a warning.
		// "creating new IE from data record field in IANA IE Namespace 0, this might not be intended"
		ie.EnterpriseId = 0
	}

	if f := dr.getFieldByName(0, "informationElementId"); f != nil {
		id, ok := f.Value().(*Unsigned16)
		if !ok {
			return nil, fmt.Errorf("'informationElementId' field is not of type Unsigned16, cannot use field for deriving new IE")
		}

		iid := id.Value().(uint16)
		ie.Id = iid
	} else {
		// ID field for IE is NOT optional, so we definitely need it to be able to extract a new IE
		return nil, fmt.Errorf("cannot derive a new IE without informationElementId being present in the data record")
	}

	if f := dr.getFieldByName(0, "informationElementName"); f != nil {
		n, ok := f.Value().(*String)
		if !ok {
			return nil, fmt.Errorf("'informationElementName' field is not of type String, cannot use field for deriving new IE")
		}
		ie.Name = n.Value().(string)
	} else {
		return nil, fmt.Errorf("rejecting field with undefined name")
	}

	if f := dr.getFieldByName(0, "informationElementDescription"); f != nil {
		n, ok := f.Value().(*String)
		if ok {
			desc := n.Value().(string)
			ie.Description = &desc
		}
	}

	if f := dr.getFieldByName(0, "informationElementDataType"); f != nil {
		// DataType is given numerically, so we need to map the number to a known data type
		dt, ok := f.Value().(*Unsigned8)
		if !ok {
			return nil, fmt.Errorf("'informationElementDataType' field is not of type Unsigned8, cannot use field for deriving new IE")
		}
		dtc := DataTypeFromNumber(dt.Value().(uint8))
		typ := dtc().Type()
		ie.Type = &typ
		ie.Constructor = dtc
	}

	if f := dr.getFieldByName(0, "informationElementSemantics"); f != nil {
		semantic := semantics.Default
		sem, ok := f.Value().(*Unsigned8)
		// semantics field has a defaulting mechanism so not being able to unwrap the field is not that problematic
		if ok {
			semantic = semantics.FromNumber(sem.Value().(uint8))
		}
		ie.Semantics = semantic
	}

	if f := dr.getFieldByName(0, "informationElementUnits"); f != nil {
		r, ok := f.Value().(*Unsigned16)
		if ok {
			// unit is specified in record and needs to be converted to literal
			u := units.FromNumber(r.Value().(uint16))
			ie.Units = &u
		}
	}

	var rang *InformationElementRange
	if f := dr.getFieldByName(0, "informationElementRangeBegin"); f != nil {
		rang = &InformationElementRange{}
		r, ok := f.Value().(*Unsigned64)
		if ok {
			// TODO(zoomoid): this is potentially lossy, because IPFIX overprovisions ranges here,
			// realistically we don't need more than 32 bit for range representation and there currently are
			// no instances where this is relevant
			rang.Low = int(r.Value().(uint64))
		}
	}
	if f := dr.getFieldByName(0, "informationElementRangeEnd"); f != nil {
		if rang == nil {
			rang = &InformationElementRange{}
		}
		r, ok := f.Value().(*Unsigned64)
		if ok {
			// TODO(zoomoid): this is potentially lossy, because IPFIX overprovisions ranges here,
			// realistically we don't need more than 32 bit for range representation and there currently are
			// no instances where this is relevant
			rang.High = int(r.Value().(uint64))
		}
	}
	if rang != nil {
		ie.Range = rang
	}

	// logger.V(4).Info("created new information element from data record", "ie", ie.String())
	return ie, nil
}
