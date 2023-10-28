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
	"bytes"
	"encoding/binary"
	"testing"
)

func TestSigned64(t *testing.T) {
	t.Parallel()
	t.Run("default length", func(t *testing.T) {
		t.Parallel()
	})
	t.Run("reduced length", func(t *testing.T) {
		t.Parallel()

		t.Run("7-byte (-1)", func(t *testing.T) {
			inInt64 := int64(-1)
			// two's complement of -1 is 0xFFFFFFFF..
			in := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
			v := NewSigned64().WithLength(7)()
			_, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
		t.Run("7-byte (-12621359)", func(t *testing.T) {
			inInt64 := int64(-12621359)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(inInt64))
			in := b[1:8]
			v := NewSigned64().WithLength(7)()
			_, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
		t.Run("7-byte (162)", func(t *testing.T) {
			inInt64 := int64(162)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(inInt64))
			in := b[1:8]
			v := NewSigned64().WithLength(7)()
			_, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
	})
}
