package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestVariableField(t *testing.T) {
	t.Run("variable-string", func(t *testing.T) {
		raw := []byte{}

		raw = append(raw, 0xFF)
		raw = binary.BigEndian.AppendUint16(raw, 3)
		raw = append(raw, []byte("hi!")...)

		tc := NewDefaultEphemeralCache()

		f := NewFieldBuilder(InformationElement{
			Id:          0,
			Constructor: NewString,
		}).
			SetLength(FieldVariableLength).
			SetTemplateManager(tc).
			SetFieldManager(NewEphemeralFieldCache(tc)).
			Complete()

		n, err := f.Decode(bytes.NewBuffer(raw))
		t.Log(f.Value())
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		if n != len(raw) {
			t.Error(fmt.Errorf("not the right amount of bytes decoded in Decode, expected %d, found %d", len(raw), n))
			t.Fail()
		}
	})
}
