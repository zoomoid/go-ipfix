package ipfix

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

var (
	penMask = uint16(0x8000)
)

type BasicList struct {
	isVariableLength bool

	semantic ListSemantic

	fieldId uint16

	isEnterprise bool

	elementLength uint16

	pen uint32

	// length is the number of bytes of elements contained in a basic list.
	// Note that if the data type was created synthetically, i.e., not from decoding IPFIX
	// packets, the length value may be 0, even though the list contains elements.
	length uint16

	value []Field

	fieldManager FieldCache
}

func NewBasicList() DataType {
	return &BasicList{
		// explicitly initialize this as undefined, because the zero value of ListSemantic
		// is "noneOf" by the definition of IANA, and working around this by relabeling the
		// fields is too much of a hassle
		semantic: SemanticUndefined,
	}
}

func (t *BasicList) WithManager(mgr FieldCache) DataTypeConstructor {
	return func() DataType {
		return &BasicList{
			fieldManager: mgr,
			semantic:     SemanticUndefined,
		}
	}
}

func (t *BasicList) String() string {
	if t.value == nil {
		return "nil"
	}
	s := make([]string, len(t.value))
	for i, el := range t.value {
		s[i] = el.Value().String()
	}
	return "[" + strings.Join(s, " ") + "]"
}

func (*BasicList) Type() string {
	return "basicList"
}

func (t *BasicList) Value() interface{} {
	return t.value
}

// NOTE that this allows for various types of list items, as long as they implement
// the DataType interface; in IPFIX, this is forbidden: basicList elements must all
// have the same type, encoded by the fieldId read in the "header" bytes of the list.
//
// SetValue does not perform additional type checks as it is fine with variable types.
func (t *BasicList) SetValue(v any) DataType {
	// TODO(zoomoid): in regular IPFIX, a basicList may only contain elements of the same
	// type. This is not enforced here, any type implementing the DataType interface may
	// be passed in, and thus, types with different lengths CAN occur when using this function
	b, ok := v.([]Field)
	if !ok {
		panic(fmt.Errorf("%T cannot be asserted to %T", v, t.value))
	}

	firstType := reflect.TypeOf(b[0])
	for _, value := range b {
		if reflect.TypeOf(value) != firstType {
			panic(fmt.Errorf("basicList items are not all of the same type, expected %s, found %T", firstType.String(), value))
		}
	}

	t.value = b
	l := uint16(0)
	for _, e := range b {
		l += e.Length()
	}
	t.length = l
	return t
}

var (
	basicListMinimumHeaderLength uint16 = 1 + 2 + 2 // semantics (uint8) + fieldId (uint16) + element length (uint16)
)

func (t *BasicList) Length() uint16 {
	lh := basicListMinimumHeaderLength
	if t.isEnterprise {
		lh += 4 // pen is uint32
	}
	var length uint16
	for _, f := range t.value {
		length += f.Length()
	}
	return lh + length
}

func (t *BasicList) Clone() DataType {
	dv := make([]Field, 0)
	for _, el := range t.value {
		dv = append(dv, el.Clone())
	}
	return &BasicList{
		value:            dv,
		isVariableLength: t.isVariableLength,
		semantic:         t.semantic,
		fieldId:          t.fieldId,
		isEnterprise:     t.isEnterprise,
		length:           t.length,
		pen:              t.pen,
		fieldManager:     t.fieldManager,
	}
}

func (*BasicList) DefaultLength() uint16 {
	return 0
}

func (t *BasicList) WithLength(length uint16) DataTypeConstructor {
	return func() DataType {
		return &BasicList{
			length: length,
		}
	}
}

func (t *BasicList) SetLength(length uint16) DataType {
	t.length = length
	return t
}

func (t *BasicList) IsReducedLength() bool {
	return false
}

func (t *BasicList) Decode(r io.Reader) error {
	var err error
	var fieldId uint16
	var sematic ListSemantic
	var enterpriseId uint32
	var reverse bool
	// basicList is at least 5 bytes = semantic (1 byte) + field Id (2 byte) + element length (2 byte)
	// which, in case of enterprise-specific IEs, may also be 9 = 5 + pen (4 bytes)
	var headerLength uint16 = basicListMinimumHeaderLength

	err = binary.Read(r, binary.BigEndian, &sematic)
	if err != nil {
		return fmt.Errorf("failed to read list semantic in %T, %w", t, err)
	}
	t.semantic = sematic

	var rawFieldId uint16
	err = binary.Read(r, binary.BigEndian, &rawFieldId)
	if err != nil {
		return fmt.Errorf("failed to read field id in %T, %w", t, err)
	}

	// mask the first bit which indicates a private enterprise field
	fieldId = (^penMask) & rawFieldId
	t.fieldId = fieldId

	if rawFieldId >= 0x8000 {
		// first bit is 1, therefore this is a enterprise-specific IE
		t.isEnterprise = true
	}

	err = binary.Read(r, binary.BigEndian, &t.elementLength)
	if err != nil {
		return fmt.Errorf("failed to read element length in %T, %w", t, err)
	}

	if t.isEnterprise {
		err = binary.Read(r, binary.BigEndian, &enterpriseId)
		if err != nil {
			return fmt.Errorf("failed to read pen in %T, %w", t, err)
		}
		t.pen = enterpriseId
		if enterpriseId == ReversePEN && Reversible(fieldId) {
			reverse = true
			// clear enterprise id, because this would obscure lookup
			enterpriseId = 0
		}

		headerLength += 4
	}

	fieldBuilder, err := t.fieldManager.Get(context.TODO(), NewFieldKey(enterpriseId, fieldId))
	if err != nil {
		return fmt.Errorf("failed to get field (%d,%d) from manager in %T, %w", enterpriseId, fieldId, t, err)
	}

	if fieldBuilder == nil {
		return fmt.Errorf("undefined field id (%d,%d)", enterpriseId, fieldId)
	}

	field := fieldBuilder.
		FieldManager(t.fieldManager).
		Length(t.elementLength). // if this is 0xFFFF, this makes a VariableLengthField
		PEN(enterpriseId).
		Reverse(reverse).
		Complete()

	t.value = make([]Field, 0)
	// TODO(zoomoid): check if this is semantically equivalent!
	buf := make([]byte, t.length-headerLength)
	// buf := make([]byte, t.elementLength)

	_, err = r.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read basicList content, %w", err)
	}
	basicListContent := bytes.NewBuffer(buf)
	for i := 0; basicListContent.Len() > 0; i++ {
		err = field.Decode(basicListContent)
		if err != nil /* && !errors.Is(err, io.EOF) */ {
			return fmt.Errorf("error while decoding list element %d in %T, %w", i, t, err)
		}
		t.value = append(t.value, field)
	}

	return nil
}

func (t *BasicList) Encode(w io.Writer) (n int, err error) {
	// header
	b := make([]byte, 0)
	b = append(b, byte(t.semantic))
	if t.isEnterprise {
		b = binary.BigEndian.AppendUint16(b, penMask|t.fieldId)
	} else {
		b = binary.BigEndian.AppendUint16(b, t.fieldId)
	}
	b = binary.BigEndian.AppendUint16(b, t.elementLength)
	if t.isEnterprise {
		b = binary.BigEndian.AppendUint32(b, t.pen)
	}

	n, err = w.Write(b)
	if err != nil {
		return
	}

	for _, el := range t.Elements() {
		fn, err := el.Encode(w)
		n += fn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (t *BasicList) Semantic() ListSemantic {
	return t.semantic
}

func (t *BasicList) SetSemantic(s ListSemantic) *BasicList {
	t.semantic = s
	return t
}

func (t *BasicList) FieldID() uint16 {
	return t.fieldId
}

func (t *BasicList) SetFieldID(s uint16) *BasicList {
	t.fieldId = s
	return t
}

// Elements does the same thing as Value(), returning t.value, which is a slice of DataType-
// implementors, but with a narrower type than Values(), which returns interface{}
func (t *BasicList) Elements() []Field {
	return t.value
}

// unmarshalledDataValue is the intermediate type used for marshalling a
// basic list item to JSON or YAML. It denotes a value *not yet* being
// marshalled.
type unmarshalledDataValue struct {
	Value any    `json:"value,omitempty" yaml:"value,omitempty"`
	Type  string `json:"type,omitempty" yaml:"type,omitempty"`
}

// basicListMetadata is the intermediate type used for marshalling a basic
// list to JSON or YAML. It contains "header information" like semantic
// and list length in a structure, rather than just subsequent bytes like
// in binary IPFIX
type basicListMetadata struct {
	Semantic ListSemantic `json:"semantic" yaml:"semantic"`
	FieldId  uint16       `json:"field_id" yaml:"fieldId"`
	Length   uint16       `json:"length,omitempty" yaml:"length,omitempty"`
	PEN      uint32       `json:"pen" yaml:"pen"`

	// no need to capture isVariableLength in here, because the wrapping Field type
	// will also contain the attribute on a higher-level
}

// unmarshalledBasicList is the intermediate wrapper type for marshalling an entire
// basicList to JSON or YAML. It wraps metadata and the list of elements in a struct.
type unmarshalledBasicList struct {
	Metadata basicListMetadata       `json:"metadata" yaml:"metadata"`
	Elements []unmarshalledDataValue `json:"elements" yaml:"elements"`
}

type marshalledDataValue struct {
	// Value is any json-encoded value, so we can use it to call json.Unmarshal on
	Value json.RawMessage `json:"value,omitempty" yaml:"value,omitempty"`
	Type  string          `json:"type,omitempty" yaml:"type,omitempty"`
}

type marshalledBasicList struct {
	Metadata basicListMetadata     `json:"metadata" yaml:"metadata"`
	Elements []marshalledDataValue `json:"elements,omitempty" yaml:"elements,omitempty"`
}

func (t *BasicList) MarshalJSON() ([]byte, error) {
	ff := make([]unmarshalledDataValue, 0, len(t.value))
	for _, el := range t.value {
		ff = append(ff, unmarshalledDataValue{
			Value: el,
			Type:  el.Type(),
		})
	}

	return json.Marshal(unmarshalledBasicList{
		Metadata: basicListMetadata{
			Semantic: t.semantic,
			FieldId:  t.fieldId,
			Length:   t.Length(),
			PEN:      t.pen,
		},
		Elements: ff,
	})
}

func (t *BasicList) UnmarshalJSON(in []byte) error {
	ff := &marshalledBasicList{}
	err := json.Unmarshal(in, &ff)
	if err != nil {
		return err
	}

	t.fieldId = ff.Metadata.FieldId
	t.pen = ff.Metadata.PEN
	if t.pen != 0 {
		t.isEnterprise = true
	}
	t.length = ff.Metadata.Length + 3 // 3 bytes for semantics + fieldId
	if t.isEnterprise {
		t.length += 4 // 4 bytes for PEN
	}
	t.semantic = ff.Metadata.Semantic

	fs := make([]Field, 0, len(ff.Elements))
	for _, el := range ff.Elements {
		v := NewFieldBuilder(InformationElement{
			Constructor: LookupConstructor(el.Type),
		}).Complete()
		err := v.UnmarshalJSON(el.Value)
		if err != nil {
			return err
		}
		fs = append(fs, v)
	}
	t.value = fs

	return nil
}

func (t *BasicList) NewBuilder() ListTypeBuilder {
	return &basicListBuilder{}
}

type basicListBuilder struct {
	fieldManager FieldCache
}

func (t *basicListBuilder) WithFieldManager(fieldManager FieldCache) ListTypeBuilder {
	t.fieldManager = fieldManager
	return t
}

func (t *basicListBuilder) Complete() DataTypeConstructor {
	return func() DataType {
		return &BasicList{
			fieldManager: t.fieldManager,
		}
	}
}

var _ ListType = &BasicList{}

var _ DataTypeConstructor = NewBasicList
