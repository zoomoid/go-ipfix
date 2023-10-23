package ipfix

import (
	"github.com/zoomoid/go-ipfix/iana/semantics"
	"github.com/zoomoid/go-ipfix/iana/status"
)

func NewUnassignedFieldBuilder(id uint16) *FieldBuilder {
	return NewFieldBuilder(InformationElement{
		Name:         "unassigned",
		Id:           id,
		EnterpriseId: 0,
		Constructor:  NewOctetArray,
		Semantics:    semantics.Undefined,
		Status:       status.Undefined,
	})
}
