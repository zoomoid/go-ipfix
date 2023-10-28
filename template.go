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
	"fmt"
	"io"
	"time"
)

type TemplateMetadata struct {
	Name                string            `json:"name,omitempty"`
	TemplateId          uint16            `json:"template_id,omitempty"`
	ObservationDomainId uint32            `json:"observation_domain_id,omitempty"`
	CreationTimestamp   time.Time         `json:"created"`
	Labels              map[string]string `json:"labels,omitempty"`
	Annotations         map[string]string `json:"annotations,omitempty"`
}

type Template struct {
	*TemplateMetadata `json:"metadata,omitempty"`
	Record            templateRecord

	templateCache TemplateCache
	fieldCache    FieldCache
}

// TemplateRecord is the interface that TemplateRecord and OptionsTemplateRecord need to implement
type templateRecord interface {
	json.Marshaler
	json.Unmarshaler

	Type() string
	Id() uint16

	Encode(io.Writer) (int, error)

	DecodeData(io.Reader) (int, error)
}

func (tr *Template) WithFieldCache(f FieldCache) *Template {
	tr.fieldCache = f
	return tr
}

func (tr *Template) WithTemplateCache(f TemplateCache) *Template {
	tr.templateCache = f
	return tr
}

var _ json.Marshaler = &Template{}
var _ json.Unmarshaler = &Template{}

func (tr Template) MarshalJSON() ([]byte, error) {
	type itr struct {
		Kind     string            `json:"kind"`
		Metadata *TemplateMetadata `json:"metadata,omitempty"`
		Record   json.RawMessage   `json:"record"`
	}

	ot := itr{}

	switch t := tr.Record.(type) {
	case *TemplateRecord, *OptionsTemplateRecord:
		ot.Kind = t.Type()
		b, err := t.MarshalJSON()
		if err != nil {
			return nil, err
		}
		ot.Record = b
		return json.Marshal(ot)
	default:
		return nil, fmt.Errorf("cannot use %T as template for templates.Template", t)
	}
}

func (t *Template) UnmarshalJSON(in []byte) error {
	type itr struct {
		Kind              string `json:"kind"`
		*TemplateMetadata `json:"metadata,omitempty"`
		Record            json.RawMessage `json:"record"`
	}

	it := itr{}

	err := json.Unmarshal(in, &it)
	if err != nil {
		return nil
	}
	switch it.Kind {
	case KindTemplateRecord:
		tr := TemplateRecord{
			FieldManager:    t.fieldCache,
			TemplateManager: t.templateCache,
		}
		err := json.Unmarshal(it.Record, &tr)
		if err != nil {
			return err
		}
		t.Record = &tr
	case KindOptionsTemplateRecord:
		otr := OptionsTemplateRecord{
			FieldManager:    t.fieldCache,
			TemplateManager: t.templateCache,
		}
		err := json.Unmarshal(it.Record, &otr)
		if err != nil {
			return err
		}
		t.Record = &otr
	default:
		return fmt.Errorf("cannot use %v as a template for unmarshaling", it.Record)
	}
	return nil
}
