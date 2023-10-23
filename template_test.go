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
	"encoding/json"
	"testing"
)

var templates []Template = []Template{
	{
		Record: &TemplateRecord{
			TemplateId: 300,
			Fields: []Field{
				NewFieldBuilder(IPFIX()[2]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[150]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[10]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[14]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[4]).Length(1).Complete(),
				NewFieldBuilder(IPFIX()[6]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[1]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[7]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[11]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[8]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[12]).Length(4).Complete(),
			},
		},
	},
	{
		Record: &TemplateRecord{
			TemplateId: 501,
			Fields: []Field{
				NewFieldBuilder(IPFIX()[14]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[4]).Length(1).Complete(),
				NewFieldBuilder(IPFIX()[6]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[1]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[7]).Length(2).Complete(),
			},
		},
	},
	{
		Record: &OptionsTemplateRecord{
			TemplateId: 1591,
			Scopes: []Field{
				NewFieldBuilder(IPFIX()[346]).Length(4).Complete(),
				NewFieldBuilder(IPFIX()[303]).Length(2).Complete(),
			},
			Options: []Field{
				NewFieldBuilder(IPFIX()[339]).Length(1).Complete(),
				NewFieldBuilder(IPFIX()[344]).Length(1).Complete(),
				NewFieldBuilder(IPFIX()[345]).Length(2).Complete(),
				NewFieldBuilder(IPFIX()[342]).Length(8).Complete(),
				NewFieldBuilder(IPFIX()[343]).Length(8).Complete(),
				NewFieldBuilder(IPFIX()[341]).Length(FieldVariableLength).Complete(),
				NewFieldBuilder(IPFIX()[340]).Length(FieldVariableLength).Complete(),
			},
		},
	},
}

var marshalledTemplates [][]byte = [][]byte{
	[]byte(`{"kind":"TemplateRecord","record":{"template_id":300,"fields":[{"id":2,"name":"packetDeltaCount","length":4,"type":"unsigned64"},{"id":150,"name":"flowStartSeconds","length":4,"type":"dateTimeSeconds"},{"id":10,"name":"ingressInterface","length":2,"type":"unsigned32"},{"id":14,"name":"egressInterface","length":2,"type":"unsigned32"},{"id":4,"name":"protocolIdentifier","length":1,"type":"unsigned8"},{"id":6,"name":"tcpControlBits","length":2,"type":"unsigned16"},{"id":1,"name":"octetDeltaCount","length":4,"type":"unsigned64"},{"id":7,"name":"sourceTransportPort","length":2,"type":"unsigned16"},{"id":11,"name":"destinationTransportPort","length":2,"type":"unsigned16"},{"id":8,"name":"sourceIPv4Address","length":4,"type":"ipv4Address"},{"id":12,"name":"destinationIPv4Address","length":4,"type":"ipv4Address"}]}}`),
	[]byte(`{"kind":"TemplateRecord","record":{"fields":[{"id":14,"name":"egressInterface","length":2,"type":"unsigned32"},{"id":4,"name":"protocolIdentifier","length":1,"type":"unsigned8"},{"id":6,"name":"tcpControlBits","length":2,"type":"unsigned16"},{"id":1,"name":"octetDeltaCount","length":4,"type":"unsigned64"},{"id":7,"name":"sourceTransportPort","length":2,"type":"unsigned16"}]}}`),
	[]byte(`{"kind":"OptionsTemplateRecord","record":{"scopes":[{"id":346,"name":"privateEnterpriseNumber","length":4,"type":"unsigned32"},{"id":303,"name":"informationElementId","length":2,"type":"unsigned16"}],"options":[{"id":339,"name":"informationElementDataType","length":1,"type":"unsigned8"},{"id":344,"name":"informationElementSemantics","length":1,"type":"unsigned8"},{"id":345,"name":"informationElementUnits","length":2,"type":"unsigned16"},{"id":342,"name":"informationElementRangeBegin","length":8,"type":"unsigned64"},{"id":343,"name":"informationElementRangeEnd","length":8,"type":"unsigned64"},{"id":341,"name":"informationElementName","length":65535,"is_variable_length":true,"type":"string"},{"id":340,"name":"informationElementDescription","length":65535,"is_variable_length":true,"type":"string"}]}}`),
}

func TestTemplate(t *testing.T) {
	t.Run("marshal template to json", func(t *testing.T) {
		for _, tt := range templates {
			b, err := json.Marshal(tt)
			if err != nil {
				t.Fatal(err)
			}

			t.Log(string(b))
		}
	})

	t.Run("unmarshal template from json", func(t *testing.T) {
		for _, tt := range marshalledTemplates {
			template := Template{}
			err := json.Unmarshal(tt, &template)
			if err != nil {
				t.Fatal(err)
			}

			t.Log(template)
		}
	})
}
