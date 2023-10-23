package ipfix

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type MacAddress struct {
	value net.HardwareAddr
}

func NewMacAddress() DataType {
	return &MacAddress{}
}

func (t *MacAddress) String() string {
	return t.value.String()
}

func (*MacAddress) Type() string {
	return "macAddress"
}

func (t *MacAddress) Value() interface{} {
	return t.value
}

func (t *MacAddress) SetValue(v any) DataType {
	switch b := v.(type) {
	case string:
		ma, err := net.ParseMAC(b)
		if err != nil {
			panic(fmt.Errorf("cannot set value in %T, %w", t, err))
		}
		t.value = ma
	case net.HardwareAddr:
		t.value = b
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T in %T", v, t.value, t))
	}
	return t
}

func (t *MacAddress) Length() uint16 {
	return t.DefaultLength()
}

func (*MacAddress) DefaultLength() uint16 {
	return 6
}

func (t *MacAddress) Clone() DataType {
	return &MacAddress{
		value: t.value,
	}
}

func (t *MacAddress) WithLength(length uint16) DataTypeConstructor {
	return NewMacAddress
}

func (t *MacAddress) SetLength(length uint16) DataType {
	// no-op because address types are always fixed-length
	return t
}

func (*MacAddress) IsReducedLength() bool {
	return false
}

func (t *MacAddress) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	// octs := make([]string, len(b))
	// for i, oct := range b {
	// 	octs[i] = fmt.Sprintf("%d", uint8(oct))
	// }
	t.value = net.HardwareAddr(b)
	// mac, err := net.ParseMAC(strings.Join(octs, ":"))
	// if err != err {
	// 	return fmt.Errorf("failed to parse MAC from string %s in %T, %w", t, string(b), err)
	// }
	// t.value = mac
	return nil
}

func (t *MacAddress) Encode(w io.Writer) (int, error) {
	return w.Write([]byte(t.value))
}

func (t *MacAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value.String())
}

func (t *MacAddress) UnmarshalJSON(in []byte) error {
	var m string
	err := json.Unmarshal(in, &m)
	if err != nil {
		return err
	}
	mac, err := net.ParseMAC(m)
	if err != nil {
		return err
	}
	t.value = mac
	return nil
}

var _ DataTypeConstructor = NewMacAddress
var _ DataType = &MacAddress{}
