package ipfix

import (
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

type FieldExport struct {
	Name            string
	ExportTimestamp time.Time

	Fields []InformationElement
}

func MustReadYAML(r io.Reader) map[uint16]InformationElement {
	m, err := ReadYAML(r)
	if err != nil {
		panic(err)
	}
	return m
}

func ReadYAML(r io.Reader) (map[uint16]InformationElement, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	read := FieldExport{}
	err := dec.Decode(&read)
	if err != nil {
		return nil, err
	}

	fields := make(map[uint16]InformationElement)

	for _, el := range read.Fields {
		id := el.Id
		fields[uint16(id)] = el
	}

	return fields, nil
}

func MustWriteYAML(w io.Writer, m map[uint16]InformationElement) {
	err := WriteYAML(w, m)
	if err != nil {
		panic(err)
	}
}

func WriteYAML(w io.Writer, m map[uint16]InformationElement) error {
	fields := make([]InformationElement, 0, len(m))

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
