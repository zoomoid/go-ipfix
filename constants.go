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
	ianaIpfixIEs map[uint16]InformationElement

	//go:embed  hack/ipfix-information-elements.csv
	spec embed.FS
)

func init() {
	iif, _ := spec.ReadFile("ipfix-information-elements.csv")
	ib := bytes.NewBuffer(iif)

	ianaIpfixIEs = MustReadCSV(ib)
}

func IANA() map[uint16]InformationElement {
	return ianaIpfixIEs
}
