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

func TestMacAddress(t *testing.T) {
	raw := []byte{0xac, 0x74, 0xb1, 0x88, 0x3a, 0xa5}

	mac := &MacAddress{}
	m, err := mac.Decode(bytes.NewBuffer(raw))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(mac.String())

	b := &bytes.Buffer{}
	n, err := mac.Encode(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, b.Bytes()) && m == len(raw) && n == len(raw) {
		t.Error("expected encoded bytes to be equal to input bytes")
	}
}
