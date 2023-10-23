package ipfix

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

func MustReadCSV(r io.Reader) map[uint16]InformationElement {
	m, err := ReadCSV(r)
	if err != nil {
		panic(err)
	}
	return m
}

func ReadCSV(r io.Reader) (map[uint16]InformationElement, error) {
	csvReader := csv.NewReader(r)

	_, _ = csvReader.Read()

	fieldMap := make(map[uint16]InformationElement)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		field := InformationElement{}

		id, _ := strconv.Atoi(record[0])
		field.Id = uint16(id)

		field.Name = record[1]

		if typ := record[2]; typ != "" {
			field.Type = &typ
			field.Constructor = LookupConstructor(typ)
		}

		if sem := record[3]; sem != "" {
			field.Semantics.UnmarshalText([]byte(sem))
		}

		if stat := record[4]; stat != "" {
			field.Status.UnmarshalText([]byte(stat))
		}
		if description := record[5]; description != "" {
			field.Description = &description
		}

		if units := record[6]; units != "" {
			field.Units = &units
		}

		fr := strings.Split(record[7], "-")
		if len(fr) == 2 {
			lows, highs := fr[0], fr[1]
			var low, high int
			if strings.HasPrefix(lows, "0x") {
				l, _ := strconv.ParseInt(lows, 16, 32)
				low = int(l)
			} else {
				low, _ = strconv.Atoi(lows)
			}
			if strings.HasPrefix(highs, "0x") {
				h, _ := strconv.ParseInt(highs, 16, 32)
				high = int(h)
			} else {
				high, _ = strconv.Atoi(highs)
			}
			field.Range = &InformationElementRange{
				Low:  low,
				High: high,
			}
		}

		if additionalInformation := record[8]; additionalInformation != "" {
			field.AdditionalInformation = &additionalInformation
		}

		if ref := record[9]; ref != "" {
			field.Reference = &ref
		}

		if revision := record[10]; revision != "" {
			rev, _ := strconv.Atoi(record[9])
			field.Revision = &rev
		}

		fieldMap[uint16(id)] = field
	}

	return fieldMap, nil
}
