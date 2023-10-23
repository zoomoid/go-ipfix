package ipfix

import (
	"bytes"
	"testing"
)

func TestUnsigned64(t *testing.T) {
	t.Parallel()
	t.Run("default length", func(t *testing.T) {
		t.Parallel()

	})
	t.Run("reduced length", func(t *testing.T) {
		t.Parallel()
		t.Run("7-byte", func(t *testing.T) {
			inUint64 := uint64(0xAB32131FFA4192)
			in := []byte{0xAB, 0x32, 0x13, 0x1F, 0xFA, 0x41, 0x92}
			v := NewUnsigned64().WithLength(7)()
			err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(uint64) != inUint64 {
				t.Errorf("expected value to be %d, found %d", inUint64, v.Value().(uint64))
			}

		})
		t.Run("6-byte", func(t *testing.T) {
			inUint64 := uint64(0xAB32131FFA41)
			in := []byte{0xAB, 0x32, 0x13, 0x1F, 0xFA, 0x41}
			v := NewUnsigned64().WithLength(uint16(len(in)))()
			err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(uint64) != inUint64 {
				t.Errorf("expected value to be %d, found %d", inUint64, v.Value().(uint64))
			}
		})
		t.Run("4-byte", func(t *testing.T) {

		})
		t.Run("3-byte", func(t *testing.T) {

		})
		t.Run("2-byte", func(t *testing.T) {

		})
		t.Run("1-byte", func(t *testing.T) {

		})
	})
}
