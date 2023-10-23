package ipfix

import (
	"testing"
)

func TestFieldConsolidate(t *testing.T) {

	t.Run("FixedLengthField", func(t *testing.T) {
		f := &FixedLengthField{
			id:          42,
			name:        "Test Field",
			pen:         0,
			constructor: NewUnsigned32,
			value: &Unsigned32{
				value: 420,
			},
		}
		t.Run("marshal", func(t *testing.T) {
			out, err := f.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			t.Log(string(out))
		})

		t.Run("unmarshal", func(t *testing.T) {
			t.Log(f.Value())

			out, err := f.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			ff := &FixedLengthField{}
			err = ff.UnmarshalJSON(out)
			if err != nil {
				t.Fatal(err)
			}

			t.Log(ff.Value())
		})

	})

	t.Run("VariableLengthField", func(t *testing.T) {

		f := &VariableLengthField{
			id:          123,
			name:        "Test Octet Array Field",
			pen:         0,
			constructor: NewOctetArray,
			value: &OctetArray{
				value: []byte{
					0x15, 0xf1, 0x64, 0x13, 0x69, 0x32,
				},
			},
		}
		t.Run("marshal", func(t *testing.T) {
			out, err := f.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			t.Log(string(out))
		})

		t.Run("unmarshal", func(t *testing.T) {
			t.Log(f.Value())
			out, err := f.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			ff := &VariableLengthField{}
			err = ff.UnmarshalJSON(out)
			if err != nil {
				t.Fatal(err)
			}

			t.Log(ff.Value())
		})
	})
}
