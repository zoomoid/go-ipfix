package ipfix

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type DateTimeMilliseconds struct {
	value time.Time
}

func NewDateTimeMilliseconds() DataType {
	return &DateTimeMilliseconds{}
}

func (t *DateTimeMilliseconds) String() string {
	return fmt.Sprintf("%v", t.value)
}

func (t DateTimeMilliseconds) Type() string {
	return "dateTimeMilliseconds"
}

func (t *DateTimeMilliseconds) Value() interface{} {
	return t.value
}

func (t *DateTimeMilliseconds) SetValue(v any) DataType {
	b, ok := v.(time.Time)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}
	t.value = b
	return t
}

func (t DateTimeMilliseconds) Length() uint16 {
	return t.DefaultLength()
}

func (t *DateTimeMilliseconds) DefaultLength() uint16 {
	return 8
}

func (t *DateTimeMilliseconds) Clone() DataType {
	return &DateTimeMilliseconds{
		value: t.value,
	}
}

// WithLength for DateTimeMilliseconds returns the default constructor, as time
// abstract data types are not reduced-length-encodable
func (*DateTimeMilliseconds) WithLength(length uint16) DataTypeConstructor {
	return NewDateTimeMilliseconds
}

func (t *DateTimeMilliseconds) SetLength(length uint16) DataType {
	// no-op because time types are always fixed-length
	return t
}

// IsReducedLength for DateTimeMilliseconds returns false, as time abstract data
// types are not reduced-length-encodable
func (*DateTimeMilliseconds) IsReducedLength() bool {
	return false
}

func (t *DateTimeMilliseconds) Decode(in io.Reader) error {
	b := make([]byte, t.Length())
	_, err := in.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read data in %T, %w", t, err)
	}
	milliseconds := binary.BigEndian.Uint64(b)
	t.value = time.UnixMilli(int64(milliseconds))
	return nil
}

func (t *DateTimeMilliseconds) Encode(w io.Writer) (int, error) {
	b := make([]byte, 0)
	b = binary.BigEndian.AppendUint64(b, uint64(t.value.UnixMilli()))
	return w.Write(b)
}

func (t *DateTimeMilliseconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value)
}

func (t *DateTimeMilliseconds) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.value)
}

var _ DataTypeConstructor = NewDateTimeMilliseconds
var _ DataType = &DateTimeMilliseconds{}
