package ipfix

import (
	"encoding/json"
	"fmt"
	"io"
)

type Boolean struct {
	value bool
}

func NewBoolean() DataType {
	return &Boolean{}
}

func (t *Boolean) String() string {
	return fmt.Sprintf("%v", bool(t.value))
}

func (Boolean) Type() string {
	return "boolean"
}

func (t *Boolean) Value() interface{} {
	return t.value
}

func (t *Boolean) SetValue(v any) DataType {
	b, ok := v.(bool)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	return t
}

func (t *Boolean) Length() uint16 {
	return t.DefaultLength()
}

func (*Boolean) DefaultLength() uint16 {
	return 1
}

func (t *Boolean) Clone() DataType {
	return &Boolean{
		value: t.value,
	}
}

// WithLength for Booleans returns the default constructor, as boolean
// abstract data types are not reduced-length encodable
func (*Boolean) WithLength(length uint16) DataTypeConstructor {
	return NewBoolean
}

func (t *Boolean) SetLength(length uint16) DataType {
	// no-op because booleans types are always fixed-length
	return t
}

// IsReducedLength for Booleans returns false, as booleans are not reduced-length-encodable
func (*Boolean) IsReducedLength() bool {
	return false
}

// Decode takes a set of bytes (specifically, SHOULD just one) and decodes it to
// a boolean information element. If in contains more than one byte, Decode panics
func (t *Boolean) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	v := b[0]
	if v == 1 {
		t.value = true
	} else if v == 2 {
		t.value = false
	} else {
		return fmt.Errorf("failed to decode %T, %w", t, ErrUndefinedEncoding)
	}
	return nil
}

func (t *Boolean) Encode(w io.Writer) (int, error) {
	b := make([]byte, 1)
	if t.value {
		b[0] = byte(1) // 1 maps to true
	} else {
		b[0] = byte(2) // 2 maps to false
	}
	return w.Write(b)
}

func (t *Boolean) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Boolean) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewBoolean
var _ DataType = &Boolean{}
