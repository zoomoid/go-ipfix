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

		f := NewFieldBuilder(&InformationElement{
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
