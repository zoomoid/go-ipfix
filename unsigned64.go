package ipfix

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type Unsigned64 struct {
	value uint64

	reducedLength bool
	length        uint16
}

func NewUnsigned64() DataType {
	return &Unsigned64{}
}

func (t *Unsigned64) String() string {
	return fmt.Sprintf("%v", uint64(t.value))
}

func (*Unsigned64) Type() string {
	return "unsigned64"
}

func (t *Unsigned64) Value() interface{} {
	return t.value
}

func (t *Unsigned64) SetValue(v any) DataType {
	switch ty := v.(type) {
	case float64:
		t.value = uint64(ty)
	case int:
		t.value = uint64(ty)
	default:
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	return t
}

func (t *Unsigned64) Length() uint16 {
	if t.length > 0 {
		return t.length
	}
	return t.DefaultLength()
}

func (*Unsigned64) DefaultLength() uint16 {
	return 8
}

func (t *Unsigned64) Clone() DataType {
	return &Unsigned64{
		value: t.value,
	}
}

func (t *Unsigned64) WithLength(length uint16) DataTypeConstructor {
	if length > 0 && length < t.DefaultLength() {
		return func() DataType {
			return &Unsigned64{
				reducedLength: true,
				length:        length,
			}
		}
	}
	return NewUnsigned64
}

func (t *Unsigned64) SetLength(length uint16) DataType {
	// only valid lengths for unsigned64 are reduced-length encodings
	if length > 0 && length < t.DefaultLength() {
		t.length = length
		t.reducedLength = true
	} else {
		t.length = t.DefaultLength()
	}
	return t
}

func (t *Unsigned64) IsReducedLength() bool {
	return t.reducedLength
}

func (t *Unsigned64) Decode(in io.Reader) error {
	// allocate a buffer of the (possibly reduced) length of the data type
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	if !t.reducedLength {
		// fast-track
		t.value = binary.BigEndian.Uint64(b)
		return nil
	}
	// because reduced-length encoding still preserves BigEndian, we pad the
	// internal uint64
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	// abusing golangs initialization of values with 0 here
	for i := uint16(0); i < t.length; i++ {
		c[i+offset] = b[i]
	}
	t.value = binary.BigEndian.Uint64(c)
	return nil
}

func (t *Unsigned64) Encode(w io.Writer) (int, error) {
	b := make([]byte, t.Length())
	if !t.reducedLength {
		// fast-track
		binary.BigEndian.PutUint64(b, t.value)
		return w.Write(b)
	}
	offset := t.DefaultLength() - t.Length()
	c := make([]byte, t.DefaultLength())
	binary.BigEndian.PutUint64(c, t.value)

	for i := uint16(0); i < t.length; i++ {
		b[i] = c[i+offset]
	}
	return w.Write(b)
}

func (t *Unsigned64) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *Unsigned64) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewUnsigned64
var _ DataType = &Unsigned64{}
