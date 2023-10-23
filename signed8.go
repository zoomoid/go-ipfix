package ipfix

import (
	"encoding/json"
	"fmt"
	"io"
)

type Signed8 struct {
	value int8
}

func NewSigned8() DataType {
	return &Signed8{}
}

func (t *Signed8) String() string {
	return fmt.Sprintf("%d", t.value)
}

func (*Signed8) Type() string {
	return "signed8"
}

func (t *Signed8) Value() interface{} {
	return t.value
}

func (t *Signed8) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = int8(ty)
	case int:
		t.value = int8(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Signed8) Length() uint16 {
	return t.DefaultLength()
}

func (t *Signed8) DefaultLength() uint16 {
	return 1
}

func (t *Signed8) Clone() DataType {
	return &Signed8{
		value: t.value,
	}
}

func (*Signed8) WithLength(length uint16) DataTypeConstructor {
	return NewSigned8
}

func (t *Signed8) SetLength(length uint16) DataType {
	// no-op, signed8 is already as short as we can get
	return t
}

func (*Signed8) IsReducedLength() bool {
	return false
}

func (t *Signed8) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	t.value = int8(uint8(b[0]))
	return nil
}

func (t *Signed8) Encode(w io.Writer) (int, error) {
	b := make([]byte, 1)
	b[0] = byte(uint8(t.value))
	return w.Write(b)
}

func (t *Signed8) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Signed8) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewSigned8
var _ DataType = &Signed8{}
