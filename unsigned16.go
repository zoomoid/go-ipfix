package ipfix

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type Unsigned16 struct {
	value uint16

	length        uint16
	reducedLength bool
}

func NewUnsigned16() DataType {
	return &Unsigned16{}
}

func (t *Unsigned16) String() string {
	return fmt.Sprintf("%v", uint16(t.value))
}

func (t Unsigned16) Type() string {
	return "unsigned16"
}

func (t *Unsigned16) Value() interface{} {
	return t.value
}

func (t *Unsigned16) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = uint16(ty)
	case int:
		t.value = uint16(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Unsigned16) Length() uint16 {
	if t.length > 0 && t.length < t.DefaultLength() {
		return t.length
	}
	return t.DefaultLength()
}

func (t *Unsigned16) DefaultLength() uint16 {
	return 2
}

func (t *Unsigned16) Clone() DataType {
	return &Unsigned16{
		value: t.value,
	}
}

func (t *Unsigned16) WithLength(length uint16) DataTypeConstructor {
	if length > 0 && length < t.DefaultLength() {
		return func() DataType {
			return &Unsigned16{
				reducedLength: true,
				length:        length,
			}
		}
	}
	return NewUnsigned16
}

func (t *Unsigned16) SetLength(length uint16) DataType {
	// only valid lengths for unsigned16 are reduced-length encodings
	if length > 0 && length < t.DefaultLength() {
		t.length = length
		t.reducedLength = true
	} else {
		t.length = t.DefaultLength()
	}
	return t
}

func (t *Unsigned16) IsReducedLength() bool {
	return t.reducedLength
}

func (t *Unsigned16) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	if !t.reducedLength {
		// fast-track
		t.value = binary.BigEndian.Uint16(b)
		return nil
	}
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	// abusing golangs initialization of values with 0 here
	for i := uint16(0); i < t.length; i++ {
		c[i+offset] = b[i]
	}
	t.value = binary.BigEndian.Uint16(c)
	return nil
}

func (t *Unsigned16) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	if !t.reducedLength {
		// fast-track
		binary.BigEndian.PutUint16(b, t.value)
		return w.Write(b)
	}
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	binary.BigEndian.PutUint16(c, t.value)

	for i := uint16(0); i < t.length; i++ {
		b[i] = c[i+offset]
	}
	return w.Write(b)
}

func (t *Unsigned16) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Unsigned16) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewUnsigned16
var _ DataType = &Unsigned16{}
