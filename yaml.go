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
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

type FieldExport struct {
	Name            string
	ExportTimestamp time.Time

	Fields []*InformationElement
}

func MustReadYAML(r io.Reader) map[uint16]*InformationElement {
	m, err := ReadYAML(r)
	if err != nil {
		panic(err)
	}
	return m
}

func ReadYAML(r io.Reader) (map[uint16]*InformationElement, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	read := FieldExport{}
	err := dec.Decode(&read)
	if err != nil {
		return nil, err
	}

	fields := make(map[uint16]*InformationElement)

	for _, el := range read.Fields {
		id := el.Id
		fields[uint16(id)] = el
	}

	return fields, nil
}

func MustWriteYAML(w io.Writer, m map[uint16]*InformationElement) {
	err := WriteYAML(w, m)
	if err != nil {
		panic(err)
	}
}

func WriteYAML(w io.Writer, m map[uint16]*InformationElement) error {
	fields := make([]*InformationElement, 0, len(m))

	for id, el := range m {
		el.Id = id
		fields = append(fields, el)
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	err := enc.Encode(FieldExport{
		ExportTimestamp: time.Now(),
		Name:            "IP Flow Information Export (IPFIX) Entities",
		Fields:          fields,
	})
	if err != nil {
		return err
	}

	return nil
}
