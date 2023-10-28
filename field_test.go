/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
