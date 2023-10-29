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
	"errors"
	"math"
	"testing"
)

func TestSigned64(t *testing.T) {
	t.Parallel()
	t.Run("default length", func(t *testing.T) {
		t.Parallel()

		t.Run("positive small numbers", func(t *testing.T) {
			ns := []int64{1, 2, 5, 1269, 1239126, 123961236015, 39491069495}
			for _, n := range ns {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(n))
				m, err := NewSigned64().Decode(bytes.NewBuffer(b))
				if err != nil {
					t.Error(err)
					t.Fail()
				}
				if m != 8 {
					t.Error(errors.New("read too few bytes"))
				}
			}
		})

		t.Run("positive large number", func(t *testing.T) {
			ns := []int64{169812062911239126, 969884123961236015, 23941039491069495, 1269918213496767845}
			for _, n := range ns {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(n))
				m, err := NewSigned64().Decode(bytes.NewBuffer(b))
				if err != nil {
					t.Error(err)
					t.Fail()
				}
				if m != 8 {
					t.Error(errors.New("read too few bytes"))
				}
			}
		})

		t.Run("negative large number", func(t *testing.T) {

		})

		t.Run("max int64", func(t *testing.T) {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(math.MaxInt64))
			m, err := NewSigned64().Decode(bytes.NewBuffer(b))
			if err != nil {
				t.Error(err)
				t.Fail()
			}
			if m != 8 {
				t.Error(errors.New("read too few bytes"))
			}
		})

		t.Run("min int64", func(t *testing.T) {
			b := make([]byte, 8)

			i := int64(math.MinInt64)

			binary.BigEndian.PutUint64(b, uint64(i))
			m, err := NewSigned64().Decode(bytes.NewBuffer(b))
			if err != nil {
				t.Error(err)
				t.Fail()
			}
			if m != 8 {
				t.Error(errors.New("read too few bytes"))
			}
		})

	})
	t.Run("reduced length", func(t *testing.T) {
		t.Parallel()

		t.Run("7-byte (-1)", func(t *testing.T) {
			inInt64 := int64(-1)
			// two's complement of -1 is 0xFFFFFFFF..
			in := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
			v := NewSigned64().WithLength(7)()
			n, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 && n == len(in) {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
		t.Run("7-byte (-12621359)", func(t *testing.T) {
			inInt64 := int64(-12621359)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(inInt64))
			in := b[1:8]
			v := NewSigned64().WithLength(7)()
			n, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 && n == len(in) {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
		t.Run("7-byte (162)", func(t *testing.T) {
			inInt64 := int64(162)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(inInt64))
			in := b[1:8]
			v := NewSigned64().WithLength(7)()
			n, err := v.Decode(bytes.NewBuffer(in))
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			if v.Value().(int64) != inInt64 && n == len(in) {
				t.Errorf("expected value to be %d (%0x), found %d (%x)", inInt64, inInt64, v.Value().(int64), v.Value().(int64))
			}
		})
	})
}
