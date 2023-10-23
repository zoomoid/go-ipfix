package ipfix

import (
	"errors"
	"fmt"

	"github.com/zoomoid/go-ipfix/iana/version"
)

var (
	ErrTemplateNotFound error = errors.New("template not found")
	ErrUnknownVersion   error = errors.New("unknown version")
	ErrUnknownFlowId    error = errors.New("unknown flow id")
)

func TemplateNotFound(observationDomainId uint32, templateId uint16) error {
	return fmt.Errorf("%w for %d in observation domain %d", ErrTemplateNotFound, templateId, observationDomainId)
}

func UnknownVersion(version version.ProtocolVersion) error {
	return fmt.Errorf("%w %d, only 9 and 10 are specified", ErrUnknownVersion, version)
}

func UnknownFlowId(id uint16) error {
	return fmt.Errorf("%w %d", ErrUnknownFlowId, id)
}
