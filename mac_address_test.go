package ipfix

import (
	"bytes"
	"testing"
)

func TestMacAddress(t *testing.T) {
	raw := []byte{0xac, 0x74, 0xb1, 0x88, 0x3a, 0xa5}

	mac := &MacAddress{}
	err := mac.Decode(bytes.NewBuffer(raw))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(mac.String())

	b := &bytes.Buffer{}
	_, err = mac.Encode(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, b.Bytes()) {
		t.Error("expected encoded bytes to be equal to input bytes")
	}
}
