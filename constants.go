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
	"embed"
)

var (
	//go:embed hack/ipfix-information-elements.csv
	spec embed.FS

	ianaIpfixIEs map[uint16]*InformationElement = MustReadCSV(mustReadFile(spec.ReadFile("hack/ipfix-information-elements.csv")))
)

func init() {
	initGlobalIANARegistry()
}

func initGlobalIANARegistry() {
	ianaIpfixIEs = MustReadCSV(mustReadFile(spec.ReadFile("hack/ipfix-information-elements.csv")))
}

func iana() map[uint16]*InformationElement {
	if len(ianaIpfixIEs) == 0 {
		initGlobalIANARegistry()
	}

	return ianaIpfixIEs
}

func mustReadFile(f []byte, err error) *bytes.Buffer {
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(f)
}
