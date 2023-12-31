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

import "strings"

// ReversePEN is the private enterprise number designated for signaling bidirectional flow information
// contained in the record. By default, if a field has this PEN, the prototype IE is taken from the IANA
// IEs (PEN == 0), but the semantic value of the field denotes that the information describes data "in the opposite"
// direction of the flow. The decoder handles this transparently, i.e., lookup of IEs from the FieldCache is adjusted
// for.
//
// All IANA IEs are reversible, except for the ones listed in nonReversibleFields, which is used to check if the exporter
// illegaly used the reverse PEN on a field that is not reversible.
const ReversePEN uint32 = 29305

// NonReversibleFields is the lookup map for fields that are _not_ reversible as per RFC 5103.
// Authoritative information is found in https://datatracker.ietf.org/doc/html/rfc5103
var NonReversibleFields map[uint16]InformationElement = map[uint16]InformationElement{
	// identifiers per https://datatracker.ietf.org/doc/html/rfc5102#section-5.1
	10: {
		Id:   10,
		Name: "ingressInterface",
	},
	14: {
		Id:   14,
		Name: "egressInterface",
	},
	137: {
		Id:   137,
		Name: "commonPropertiesId",
	},
	138: {
		Id:   138,
		Name: "observationPointId",
	},
	141: {
		Id:   141,
		Name: "lineCardId",
	},
	142: {
		Id:   142,
		Name: "portId",
	},
	143: {
		Id:   143,
		Name: "meteringProcessId",
	},
	144: {
		Id:   144,
		Name: "exportingProcessId",
	},
	145: {
		Id:   145,
		Name: "templateId",
	},
	148: {
		Id:   148,
		Name: "flowId",
	},
	149: {
		Id:   149,
		Name: "observationDomainId",
	},
	// process configuration per https://datatracker.ietf.org/doc/html/rfc5102#section-5.2
	130: {
		Id:   130,
		Name: "exporterIPv4Address",
	},
	131: {
		Id:   131,
		Name: "exporterIPv6Address",
	},
	217: {
		Id:   217,
		Name: "exporterTransportPort",
	},
	211: {
		Id:   211,
		Name: "collectorIPv4Address",
	},
	212: {
		Id:   212,
		Name: "collectorIPv6Address",
	},
	213: {
		Id:   213,
		Name: "exportInterface",
	},
	214: {
		Id:   214,
		Name: "exportProtocolVersion",
	},
	215: {
		Id:   215,
		Name: "exportTransportVersion",
	},
	216: {
		Id:   216,
		Name: "collecotrTransportPort",
	},
	173: {
		Id:   173,
		Name: "flowKeyIndicator",
	},
	// Metering and exporting process statistics per https://datatracker.ietf.org/doc/html/rfc5102#section-5.3
	41: {
		Id:   41,
		Name: "exportedMessageTotalCount",
	},
	40: {
		Id:   40,
		Name: "exportedOctetTotalCount",
	},
	42: {
		Id:   42,
		Name: "exportedFlowRecordTotalCount",
	},
	163: {
		Id:   163,
		Name: "observedFlowTotalCount",
	},
	164: {
		Id:   164,
		Name: "ignoredPacketTotalCount",
	},
	165: {
		Id:   165,
		Name: "ignoredOctetTotalCount",
	},
	166: {
		Id:   166,
		Name: "notSentFlowTotalCount",
	},
	167: {
		Id:   167,
		Name: "notSentPacketTotalCount",
	},
	168: {
		Id:   168,
		Name: "notSentOctetTotalCount",
	},
	// padding octets per https://datatracker.ietf.org/doc/html/rfc5102#section-5.12.1
	210: {
		Id:   210,
		Name: "paddingOctets",
	},
	// biflowDirection per https://datatracker.ietf.org/doc/html/rfc5103#section-6.3
	239: {
		Id:   239,
		Name: "biflowDirection",
	},
}

// reversible looks up an IEs membership in the list of irreversible IEs
func reversible(fieldId uint16) bool {
	_, nonReversible := NonReversibleFields[fieldId]
	return !nonReversible
}

// reversedName prefixes a field's usual name with "reversed" in camelCase
// to textually indicate the presence of PEN 29305.
//
// Note that this does only account for the bidirectionality mechanism described in
// RFC 5103. Other implementors may opt for including designated IEs for reversed and non-reversed
// information in their registries, *for which this function will NOT return a sensible name*.
func reversedName(name string) string {
	s := strings.ToUpper(string([]rune(name)[0:1])) // UTF-8
	return "reversed" + s + name[1:]
}
