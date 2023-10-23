package version

import "testing"

func TestVersionString(t *testing.T) {
	ipfixLit := IPFIX
	if s := ipfixLit.String(); s != "IPFIX" {
		t.Fatalf("expected IFPIX, found %s", s)
	}

	ipfixNum := ProtocolVersion(10)
	if s := ipfixNum.String(); s != "IPFIX" {
		t.Fatalf("expected IFPIX, found %s", s)
	}

	unknown := ProtocolVersion(0)
	if s := unknown.String(); s != "Unknown" {
		t.Fatalf("expected unknown, found %s", s)
	}

	unknown2 := ProtocolVersion(4)
	if s := unknown2.String(); s != "Unknown" {
		t.Fatalf("expected unknown, found %s", s)
	}

	unknown3 := ProtocolVersion(1)
	if s := unknown3.String(); s != "Unknown" {
		t.Fatalf("expected unknown, found %s", s)
	}
}

func TestMarshalText(t *testing.T) {
	ipfixLit := IPFIX
	if _, err := ipfixLit.MarshalText(); err != nil {
		t.Fatal(err)
	}

	unknown := ProtocolVersion(0)
	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal(err)
	}
}

func TestUnmarshalText(t *testing.T) {
	p := ProtocolVersion(0)

	if err := p.UnmarshalText([]byte("IPFIX")); err != nil {
		t.Fatal(err)
	}

	if err := p.UnmarshalText([]byte("unknown")); err == nil {
		t.Fatal(err)
	}
}
