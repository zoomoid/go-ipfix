package ipfix

import (
	"encoding/binary"
	"io"

	"github.com/zoomoid/go-ipfix/iana/version"
)

type Message struct {
	Version             version.ProtocolVersion `json:"-" yaml:"-"`
	Length              uint16                  `json:"length,omitempty" yaml:"length,omitempty"`
	ExportTime          uint32                  `json:"export_time,omitempty" yaml:"exportTime,omitempty"`
	SequenceNumber      uint32                  `json:"sequence_number,omitempty" yaml:"sequenceNumber,omitempty"`
	ObservationDomainId uint32                  `json:"observation_domain_id,omitempty" yaml:"observationDomainId,omitempty"`
	Sets                []Set                   `json:"sets,omitempty" yaml:"sets,omitempty"`
}

func (p *Message) Encode(w io.Writer) (int, error) {
	b := make([]byte, 0)

	// packet header
	b = binary.BigEndian.AppendUint16(b, uint16(p.Version))
	b = binary.BigEndian.AppendUint16(b, p.Length)
	b = binary.BigEndian.AppendUint32(b, p.ExportTime)
	b = binary.BigEndian.AppendUint32(b, p.SequenceNumber)
	b = binary.BigEndian.AppendUint32(b, p.ObservationDomainId)

	nh, err := w.Write(b)
	if err != nil {
		return nh, err
	}

	// packet payload
	var nb int
	for _, fs := range p.Sets {
		nfs, err := fs.Encode(w)
		nb += nfs
		if err != nil {
			return nb, err
		}
	}
	return nh + nb, err
}
