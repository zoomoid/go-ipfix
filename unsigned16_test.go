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
	"testing"
)

func TestUnsigned16(t *testing.T) {
	t.Run("with reduced length", func(t *testing.T) {
		dt := NewUnsigned16().SetLength(1)

		_, err := dt.Decode(bytes.NewBuffer([]byte{0x0f}))
		if err != nil {
			t.Fatal(err)
		}

		t.Log(dt.Value())
	})
}
