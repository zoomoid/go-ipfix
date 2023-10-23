package ipfix

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"github.com/zoomoid/go-ipfix/iana/semantics"
	"github.com/zoomoid/go-ipfix/iana/status"
)

func MustReadXML(r io.Reader) map[uint16]InformationElement {
	m, err := ReadXML(r)
	if err != nil {
		panic(err)
	}
	return m
}

func ReadXML(r io.Reader) (map[uint16]InformationElement, error) {
	type yafIERecord struct {
		Name string `xml:"name"`
		// colons are XML namespaces, which are denoted as spaces in struct tags
		EnterpriseId uint32             `xml:"enterpriseId"`
		Reversible   bool               `xml:"reversible"`
		Id           string             `xml:"elementId"`
		Description  []string           `xml:"description>paragraph"`
		DataType     *string            `xml:"dataType"`
		Group        *string            `xml:"group"`
		Revision     *int               `xml:"revision"`
		Status       status.Status      `xml:"status"`
		Semantic     semantics.Semantic `xml:"semantic"`
		Date         *string            `xml:"date"`
		Range        *string            `xml:"range"`
		Units        *string            `xml:"units"`
	}
	type yafIERegistry struct {
		Id      *string `xml:"id,attr"`
		Title   *string `xml:"title"`
		Created *string `xml:"created"`
		Updated *string `xml:"updated"`

		Records []yafIERecord `xml:"registry>record"`
	}

	o, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	re := yafIERegistry{}
	err = xml.Unmarshal(o, &re)
	if err != nil {
		return nil, err
	}

	m := make(map[uint16]InformationElement)

	for _, r := range re.Records {
		field := InformationElement{
			Name:         r.Name,
			Semantics:    r.Semantic,
			Status:       r.Status,
			Units:        r.Units,
			Revision:     r.Revision,
			Date:         r.Date,
			Type:         r.DataType,
			EnterpriseId: r.EnterpriseId,
		}

		if description := r.Description; description != nil {
			for idx, d := range description {
				description[idx] = strings.TrimSpace(d)
			}
			d := strings.Join(description, "\n")
			field.Description = &d
		}

		if r.Range != nil {
			if fr := strings.Split(*r.Range, "-"); len(fr) == 2 {
				lows, highs := fr[0], fr[1]
				var low, high int
				low, _ = strconv.Atoi(lows)
				high, _ = strconv.Atoi(highs)
				field.Range = &InformationElementRange{
					Low:  low,
					High: high,
				}
			}
		}

		if typ := r.DataType; typ != nil {
			field.Constructor = LookupConstructor(*typ)
		}

		if id, err := strconv.Atoi(r.Id); err != nil {
			// id node is not a single number, skipping record node
			// TODO(zoomoid): maybe warn?
			continue
		} else {
			field.Id = uint16(id)
			m[uint16(id)] = field
		}
	}

	return m, nil
}
