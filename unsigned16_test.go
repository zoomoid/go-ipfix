package ipfix

import (
	"bytes"
	"testing"
)

func TestUnsigned16(t *testing.T) {
	t.Run("with reduced length", func(t *testing.T) {
		dt := NewUnsigned16().SetLength(1)

		err := dt.Decode(bytes.NewBuffer([]byte{0x0f}))
		if err != nil {
			t.Fatal(err)
		}

		t.Log(dt.Value())
	})
}
