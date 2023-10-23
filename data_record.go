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
	"fmt"
	"io"
	"strings"

	"github.com/zoomoid/go-ipfix/iana/semantics"
	"github.com/zoomoid/go-ipfix/iana/units"
)

type DataRecord struct {
	TemplateId uint16 `json:"template_id,omitempty"`
	FieldCount uint16 `json:"field_count,omitempty"`

	Fields []Field `json:"fields,omitempty"`
}

func (dr *DataRecord) Encode(w io.Writer) (n int, err error) {
	for _, r := range dr.Fields {
		rn, err := r.Encode(w)
		n += rn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (d *DataRecord) Length() uint16 {
	l := uint16(0)
	for _, f := range d.Fields {
		l += f.Length()
	}
	return l // header bytes are included on the Set!
}

// DefinesNewInformationElement is used as a pre-check to determine if a DataRecord carries
// new a Information Element definition. Afterwards, the data record is parsed in best-effort
// strategy and the new field is added to the field manager
func (dr *DataRecord) DefinesNewInformationElements() bool {
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
	// if idField != nil && nameField != nil {
	// 	logger.V(3).Info("found IE-defining record", "template_id", dr.TemplateId, "field_id", idField.Value().String(), "field_name", nameField.Value().String())
	// }
	return idField != nil && nameField != nil
}

func (dr *DataRecord) ToInformationElement() (*InformationElement, error) {
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

// func (dr *DataRecord) getFieldById(enterpriseId uint32, fieldId uint16) Field {
// 	for _, f := range dr.Fields {
// 		if f.PEN() == enterpriseId && f.Id() == fieldId {
// 			return f
// 		}
// 	}
// 	return nil
// }

func (dr *DataRecord) getFieldByName(enterpriseId uint32, name string) Field {
	for _, f := range dr.Fields {
		if f.PEN() == enterpriseId && f.Name() == name {
			return f
		}
	}
	return nil
}

func (dr *DataRecord) String() string {
	s := []string{}
	for _, v := range dr.Fields {
		s = append(s, fmt.Sprintf("%s[%d]:%s", v.Name(), v.Id(), v.Value().String()))
	}

	return fmt.Sprintf("DataRecords[%s]", strings.Join(s, " "))
}

func (dr *DataRecord) UnmarshalJSON(in []byte) error {
	type idr struct {
		TemplateId uint16 `json:"template_id,omitempty"`
		FieldCount uint16 `json:"field_count,omitempty"`

		Fields []ConsolidatedField `json:"fields,omitempty"`
	}

	t := &idr{}

	err := json.Unmarshal(in, t)
	if err != nil {
		return err
	}

	dr.TemplateId = t.TemplateId
	dr.FieldCount = t.FieldCount
	fs := make([]Field, 0, len(t.Fields))
	for _, cf := range t.Fields {
		// TODO(zoomoid): check if this is ok, i.e., "we don't need the FieldManager and TemplateManager here anymore"
		fs = append(fs, cf.Restore(nil, nil))
	}
	dr.Fields = fs

	return nil
}

func (d *DataRecord) Clone() DataRecord {
	fs := make([]Field, 0)
	for _, f := range d.Fields {
		fs = append(fs, f.Clone())
	}

	return DataRecord{
		TemplateId: d.TemplateId,
		FieldCount: d.FieldCount,

		Fields: fs,
	}
}
