package version

import (
	"errors"
)

type ProtocolVersion uint16

var (
	ErrUnknownProtocolVersion = errors.New("unknown protocol version")
)

const (
	Unknown ProtocolVersion = 0 << iota

	IPFIX ProtocolVersion = 10
)

func (p ProtocolVersion) String() string {
	switch p {
	case IPFIX:
		return "IPFIX"
	default:
		return "Unknown"
	}
}

func (p ProtocolVersion) MarshalText() ([]byte, error) {
	s := p.String()
	if s == "Unknown" {
		return nil, ErrUnknownProtocolVersion
	}
	b := []byte(s)
	return b, nil
}

func (p *ProtocolVersion) UnmarshalText(in []byte) error {
	s := string(in)
	// unwrap JSON string delimiters

	switch s {
	case "IPFIX", "ipfix":
		*p = IPFIX
	default:
		return ErrUnknownProtocolVersion
	}
	return nil
}
