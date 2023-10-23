package ipfix

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type IPv4Address struct {
	value net.IP
}

func NewIPv4Address() DataType {
	return &IPv4Address{}
}

func (t *IPv4Address) String() string {
	return t.value.To4().String()
}

func (*IPv4Address) Type() string {
	return "ipv4Address"
}

func (t *IPv4Address) Value() interface{} {
	return t.value
}

func (t *IPv4Address) SetValue(v any) DataType {
	switch b := v.(type) {
	case string:
		t.value = net.ParseIP(b)
	case net.IP:
		t.value = b
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T in %T", v, t.value, t))
	}
	return t
}

func (t *IPv4Address) Length() uint16 {
	return t.DefaultLength()
}

func (*IPv4Address) DefaultLength() uint16 {
	return 4
}

func (t *IPv4Address) Clone() DataType {
	return &IPv4Address{
		value: t.value,
	}
}

func (*IPv4Address) WithLength(length uint16) DataTypeConstructor {
	return NewIPv4Address
}

func (t *IPv4Address) SetLength(length uint16) DataType {
	// no-op because address types are always fixed-length
	return t
}

func (*IPv4Address) IsReducedLength() bool {
	return false
}

func (t *IPv4Address) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = net.IP(b)
	return nil
}

func (t *IPv4Address) Encode(w io.Writer) (int, error) {
	return w.Write([]byte(t.value))
}

func (t *IPv4Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *IPv4Address) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewIPv4Address
var _ DataType = &IPv4Address{}
