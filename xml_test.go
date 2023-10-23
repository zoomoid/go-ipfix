package ipfix

import (
	"bytes"
	"os"
	"testing"
)

func TestReadXML(t *testing.T) {
	t.Run("with file", func(t *testing.T) {
		srcFile, _ := os.Open("./cert_ipfix.xml")
		defer srcFile.Close()

		m, err := ReadXML(srcFile)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(m)
	})

	t.Run("with buffer", func(t *testing.T) {

		buf := bytes.NewBuffer(ie)

		m, err := ReadXML(buf)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(m)
	})

}

var ie []byte = []byte(`
<registry id="cert_ipfix"
          xmlns="http://www.iana.org/assignments"
          xmlns:cert="http://www.cert.org/ipfix">

  <title>CERT IPFIX Registry</title>
  <created>2017-11-28</created>
  <updated>2022-11-01</updated>

  <registry id="cert-information-elements">
    <title>CERT Enterprise IPFIX Elements (PEN 6871)</title>

		<record>
      <name>obsoleteReversePacketTotalCount</name>
      <dataType>unsigned64</dataType>
      <group>yaf</group>
      <dataTypeSemantics>totalCounter</dataTypeSemantics>
      <cert:enterpriseId>6871</cert:enterpriseId>
      <elementId>13</elementId>
      <status>obsolete</status>
      <revision>0</revision>
    </record>
	</registry>
</registry>
`)
