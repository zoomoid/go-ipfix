package ipfix

import (
	"bytes"
	"fmt"
)

func DecodeUsingTemplate(p *bytes.Buffer, fields []Field) ([]Field, error) {
	dfs := make([]Field, 0, len(fields))
	for idx, templateField := range fields {
		// Clone the field of the template to decode the value into while also preserving the
		// template information
		tf := templateField.Clone()
		name := tf.Name()
		err := tf.Decode(p)
		if err != nil {
			return nil, fmt.Errorf("failed to decode field (%d, %d/%d [%s]), %w", idx, tf.PEN(), tf.Id(), name, err)
		}
		dfs = append(dfs, tf)
	}
	return dfs, nil
}
