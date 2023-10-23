package ipfix

import (
	"encoding/json"

	"github.com/zoomoid/go-ipfix/iana/semantics"
	"github.com/zoomoid/go-ipfix/iana/status"
)

type InformationElementRange struct {
	Low  int `json:"low,omitempty" yaml:"low,omitempty"`
	High int `json:"high,omitempty" yaml:"high,omitempty"`
}

func (i *InformationElementRange) Clone() *InformationElementRange {
	return &InformationElementRange{
		Low:  i.Low,
		High: i.High,
	}
}

type InformationElement struct {
	Constructor DataTypeConstructor `json:"-" yaml:"-"`

	Id           uint16 `json:"id,omitempty" yaml:"id,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	EnterpriseId uint32 `json:"pen,omitempty" yaml:"pen,omitempty"`

	Semantics semantics.Semantic `json:"semantics,omitempty" yaml:"semantics,omitempty"`
	Status    status.Status      `json:"status,omitempty" yaml:"status,omitempty"`

	Type                  *string                  `json:"type,omitempty" yaml:"type,omitempty"`
	Description           *string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Units                 *string                  `json:"units,omitempty" yaml:"units,omitempty"`
	Range                 *InformationElementRange `json:"range,omitempty" yaml:"range,omitempty"`
	AdditionalInformation *string                  `json:"additional_information,omitempty" yaml:"additionalInformation,omitempty"`
	Reference             *string                  `json:"reference,omitempty" yaml:"reference,omitempty"`
	Revision              *int                     `json:"revision,omitempty" yaml:"revision,omitempty"`
	Date                  *string                  `json:"date,omitempty" yaml:"date,omitempty"`
}

func (i InformationElement) String() string {
	if i.Type == nil && i.Constructor != nil {
		typ := i.Constructor().Type()
		i.Type = &typ
	}

	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (i *InformationElement) Clone() InformationElement {
	ie := InformationElement{
		Id:           i.Id,
		Name:         i.Name,
		EnterpriseId: i.EnterpriseId,
		Semantics:    i.Semantics,
		Status:       i.Status,
	}

	if i.Constructor != nil {
		ie.Constructor = i.Constructor
	}
	if i.Range != nil {
		ie.Range = i.Range.Clone()
	}
	if i.Type != nil {
		typ := *i.Type
		ie.Type = &typ
	}
	if i.Description != nil {
		desc := *i.Description
		ie.Description = &desc
	}
	if i.AdditionalInformation != nil {
		ai := *i.AdditionalInformation
		ie.AdditionalInformation = &ai
	}
	if i.Units != nil {
		u := *i.Units
		ie.Units = &u
	}
	if i.Reference != nil {
		r := *i.Reference
		ie.Reference = &r
	}
	if i.Revision != nil {
		r := *i.Revision
		ie.Revision = &r
	}
	if i.Date != nil {
		d := *i.Date
		ie.Date = &d
	}

	return ie
}

func (i *InformationElement) UnmarshalJSON(in []byte) error {
	type serializableInformationElement struct {
		Id           uint16 `json:"id,omitempty" yaml:"id,omitempty"`
		EnterpriseId uint32 `json:"pen,omitempty" yaml:"pen,omitempty"`
		Name         string `json:"name,omitempty" yaml:"name,omitempty"`

		Semantics semantics.Semantic `json:"semantics,omitempty" yaml:"semantics,omitempty"`
		Status    status.Status      `json:"status,omitempty" yaml:"status,omitempty"`

		Type                  *string                  `json:"type,omitempty" yaml:"type,omitempty"`
		Description           *string                  `json:"description,omitempty" yaml:"description,omitempty"`
		Units                 *string                  `json:"units,omitempty" yaml:"units,omitempty"`
		Range                 *InformationElementRange `json:"range,omitempty" yaml:"range,omitempty"`
		AdditionalInformation *string                  `json:"additional_information,omitempty" yaml:"additionalInformation,omitempty"`
		Reference             *string                  `json:"reference,omitempty" yaml:"reference,omitempty"`
		Revision              *int                     `json:"revision,omitempty" yaml:"revision,omitempty"`
		Date                  *string                  `json:"date,omitempty" yaml:"date,omitempty"`
	}

	ii := serializableInformationElement{}
	err := json.Unmarshal(in, &ii)
	if err != nil {
		return err
	}

	i.Id = ii.Id
	i.Name = ii.Name
	i.Description = ii.Description
	i.Semantics = ii.Semantics
	i.Status = ii.Status
	i.Type = ii.Type
	i.Range = ii.Range
	i.Date = ii.Date
	i.Units = ii.Units
	i.Reference = ii.Reference
	i.AdditionalInformation = ii.AdditionalInformation
	i.Revision = ii.Revision
	i.EnterpriseId = ii.EnterpriseId

	// if type is not defined for field, exit here
	if i.Type == nil {
		return nil
	}

	i.Constructor = LookupConstructor(*i.Type)
	return nil
}
