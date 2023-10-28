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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// Decoder is instantiated with a fieldManager and a templateManager
// such that it can decode IPFIX packets into Records containing fields
// and additionally learn new fields and templates.
type Decoder struct {
	// fieldManager stores and manages field definitions for IEs to decode into. It is injected into the decoder at creation.
	// Particularly, fieldManager is able to learn new fields from options templates and subsequent data records.
	fieldManager FieldCache

	// templateManager stores and manages templates. It is injected into the decoder at creation
	templateManager TemplateCache

	completionHook completionHook

	options DecoderOptions

	metrics *decoderMetrics
}

type DecoderOptions struct {
	OmitRFC5610Records bool
}

var (
	DefaultDecoderOptions = DecoderOptions{
		OmitRFC5610Records: false,
	}
)

func (o *DecoderOptions) Merge(opts ...DecoderOptions) {
	for _, opt := range opts {
		o.OmitRFC5610Records = o.OmitRFC5610Records || opt.OmitRFC5610Records
	}
}

type completionHook func(*decoderMetrics)

type decoderMetrics struct {
	TotalLength    int64 `json:"total_length,omitempty"`
	DecodedSets    int64 `json:"decoded_messages,omitempty"`
	DecodedRecords int64 `json:"decoded_records,omitempty"`
	DroppedRecords int64 `json:"dropped_records,omitempty"`
}

// NewDecoder creates a new Decoder for a given template cache and field manager
func NewDecoder(templates TemplateCache, fields FieldCache, opts ...DecoderOptions) *Decoder {
	options := DefaultDecoderOptions
	options.Merge(opts...)

	d := &Decoder{
		fieldManager:    fields,
		templateManager: templates,
		options:         options,
		metrics:         &decoderMetrics{},
	}

	d.initMetrics()

	return d
}

func (d *Decoder) WithCompletionHook(hook func(*decoderMetrics)) *Decoder {
	d.completionHook = hook
	return d
}

// Decode takes payload as a buffer and consumes it to construct an IPFIX packet
// containing records containing decoded fields.
func (d *Decoder) Decode(ctx context.Context, payload *bytes.Buffer) (msg *Message, err error) {
	decoderStart := time.Now()

	// update metrics at the end of decoding depending on the outcome
	defer func() {
		DurationMicroseconds.Observe(float64(time.Since(decoderStart).Nanoseconds()) / 1000) // use nanoseconds for higher precision and then convert it back to microseconds
		PacketsTotal.Inc()
		if err != nil {
			ErrorsTotal.Inc()
		}
	}()

	defer func() {
		if d.completionHook != nil {
			d.completionHook(d.metrics)
		}
		d.resetMetrics()
	}()

	if d.templateManager == nil {
		return nil, errors.New("used decoder before template cache was initialized")
	}

	n, err := msg.Decode(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to read IPFIX packet header, %w", err)
	}
	d.metrics.TotalLength += int64(n) // IPFIX header length

	for i := 1; payload.Len() > 0; i++ {
		// set decoding loop
		h := SetHeader{}
		_, err := h.Decode(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to read SetHeader, %w", err)
		}
		d.metrics.TotalLength += 4
		// offset is the number of bytes in the record's payload without the
		// 4 header (2x2 bytes, templateId and set length) bytes included
		// by the protocol in the length field; binary.Size(h) captures exactly
		// that inclusion
		offset := int(h.Length) - binary.Size(h)
		if offset < 0 {
			return nil, errors.New("malformed IPFIX packet")
		}
		d.metrics.TotalLength += int64(offset)

		var set Set

		// create a fresh buffer with only the bytes of the set contents
		// TODO(zoomoid): this does some copying, and we currently cannot ensure that
		// the safety constraints of the slices are kept
		tr := bytes.NewBuffer(payload.Next(offset))

		if h.Id == IPFIX {
			// IPFIX template set
			ts := TemplateSet{
				fieldCache:    d.fieldManager,
				templateCache: d.templateManager,
			}
			_, err = ts.Decode(tr)
			if err != nil {
				return msg, fmt.Errorf("failed to decode template set at index %d, %w", i, err)
			}
			d.metrics.DecodedRecords += int64(len(ts.Records))

			set = Set{
				SetHeader: h,
				Kind:      KindTemplateRecord,
				Set:       &ts,
			}

			for _, record := range ts.Records {
				r := record // TODO(zoomoid): waiting on https://go.dev/blog/loopvar-preview
				d.templateManager.Add(ctx, TemplateKey{
					ObservationDomainId: msg.ObservationDomainId,
					TemplateId:          record.TemplateId,
				}, &Template{
					TemplateMetadata: &TemplateMetadata{
						TemplateId:          h.Id,
						ObservationDomainId: msg.ObservationDomainId,
						CreationTimestamp:   time.Now(),
					},
					Record: &r,
				})
			}
		} else if h.Id == IPFIXOptions {
			ots := &OptionsTemplateSet{
				templateCache: d.templateManager,
				fieldCache:    d.fieldManager,
			}

			// ipfix options template set
			_, err := ots.Decode(tr)
			if err != nil {
				return msg, fmt.Errorf("failed to decode options template set %d, %w", i, err)
			}
			d.metrics.DecodedRecords += int64(len(ots.Records))

			set = Set{
				SetHeader: h,
				Kind:      KindOptionsTemplateRecord,
				Set:       ots,
			}

			for _, record := range ots.Records {
				r := record // TODO(zoomoid): waiting on https://go.dev/blog/loopvar-preview
				d.templateManager.Add(ctx, TemplateKey{
					ObservationDomainId: msg.ObservationDomainId,
					TemplateId:          record.TemplateId,
				}, &Template{
					TemplateMetadata: &TemplateMetadata{
						TemplateId:          h.Id,
						ObservationDomainId: msg.ObservationDomainId,
						CreationTimestamp:   time.Now(),
					},
					Record: &r,
				})
			}
		} else if h.Id >= 256 {
			// Ids lower than 256 are reserved and not to be used for template definition
			ds := &DataSet{
				fieldCache:    d.fieldManager,
				templateCache: d.templateManager,
			}

			template, err := d.templateManager.Get(context.TODO(), TemplateKey{
				ObservationDomainId: msg.ObservationDomainId,
				TemplateId:          h.Id,
			})
			if err != nil {
				return msg, err
			}

			_, err = ds.With(template).Decode(tr)
			if err != nil {
				return msg, err
			}

			set = Set{
				SetHeader: h,
				Kind:      KindDataRecord,
				Set:       ds,
			}
		} else {
			return msg, UnknownFlowId(h.Id)
		}

		d.metrics.DecodedSets++

		DecodedSets.WithLabelValues(set.Kind).Inc()
		DecodedRecords.WithLabelValues(set.Kind).Add(float64(d.metrics.DecodedRecords))
		DroppedRecords.WithLabelValues(set.Kind).Add(float64(d.metrics.DroppedRecords))

		msg.Sets = append(msg.Sets, set)
	}

	return
}

// decodeDataSetWithTemplate decodes a data set from a given buffer reference of an IPFIX
// After decoding, the buffer is advanced by the length of the set.
// decodeDataSet returns a slice of DataRecords, or error when failing to decode.
// func decodeDataSetWithTemplate(p *bytes.Buffer, tr *TemplateRecord) ([]DataRecord, error) {
// 	records := make([]DataRecord, 0)
// 	errs := make([]error, 0)
// 	// "as long as there remains data in the buffer..."
// 	for i := 1; p.Len() > 0; i++ {
// 		fields, err := decodeUsingTemplate(p, tr.Fields)
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			errs = append(errs, fmt.Errorf("failed to decode fields of flow %d, %w", i, err))
// 			continue
// 		}
// 		record := DataRecord{
// 			Fields:     fields,
// 			TemplateId: tr.TemplateId,
// 			FieldCount: uint16(len(tr.Fields)),
// 		}
// 		// d.metrics.DecodedRecords++

// 		// check if data record is a RFC 5610 IE record, containing (new) IEs.
// 		// if not, this returns nil, nil
// 		ie, err := dataRecordToIE(record)
// 		// logger.V(2).Info("data record defined new information elements", "name", ie.Name, "id", ie.Id, "pen", ie.EnterpriseId)
// 		if err != nil {
// 			errs = append(errs, fmt.Errorf("failed to extract information element from flow %d, %w", i, err))
// 			continue
// 		}
// 		if ie != nil {
// 			err = d.fieldManager.Add(context.TODO(), *ie)
// 			if err != nil {
// 				errs = append(errs, err)
// 			}

// 		}
// 		if ie != nil && d.options.OmitRFC5610Records {
// 			d.metrics.DroppedRecords++
// 		} else {
// 			// only add records if not omitted by options
// 			records = append(records, record)
// 		}
// 	}

// 	return records, errors.Join(errs...)
// }

// decodeDataSetWithOptionsTemplate decodes a data se with a set of scopes and option fields.
// It advances the buffer by the length of the data set. decodeDataSetWithOptionsTemplate
// returns a slice of data records or an error.
//
// decodeDataSetWithOptionsTemplate does for OptionsTemplates what decodeDataSet does for regular templates.
// func decodeDataSetWithOptionsTemplate(p io.Reader, templateId uint16, scopeFields []Field, optionFields []Field) ([]DataRecord, error) {
// 	records := make([]DataRecord, 0)
// 	errs := make([]error, 0)
// 	for i := 1; ; i++ {
// 		// decode all the "scope" fields first...
// 		scopes, err := decodeUsingTemplate(p, scopeFields)
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			errs = append(errs, fmt.Errorf("failed to decode scope fields in flow %d, %w", i, err))
// 			continue
// 		}
// 		// ...then decode all the option fields
// 		options, err := decodeUsingTemplate(p, optionFields)
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			errs = append(errs, fmt.Errorf("failed to decode option fields in flow %d, %w", i, err))
// 			continue
// 		}

// 		fields := make([]Field, 0, len(scopes)+len(options))
// 		fields = append(fields, scopes...)
// 		fields = append(fields, options...)

// 		record := DataRecord{
// 			TemplateId: templateId,
// 			FieldCount: uint16(len(fields)),
// 			Fields:     fields,
// 		}
// 		d.metrics.DecodedRecords++

// 		if record.definesNewInformationElements() {
// 			ie, err := record.toInformationElement()
// 			if err != nil {
// 				errs = append(errs, fmt.Errorf("failed to extract information element from flow %d, %w", i, err))
// 				continue
// 			}
// 			if ie != nil {
// 				err = d.fieldManager.Add(context.TODO(), *ie)
// 				if err != nil {
// 					errs = append(errs, err)
// 				}
// 			}
// 			// logger.V(4).Info("data record defined new information elements", "name", ie.Name, "id", ie.Id, "pen", ie.EnterpriseId)
// 			if !d.options.OmitRFC5610Records {
// 				// only add RFC 5610 IE record if not explicitly omitted
// 				records = append(records, record)
// 			} else {
// 				d.metrics.DroppedRecords++
// 			}
// 		} else {
// 			records = append(records, record)
// 		}
// 	}

// 	return records, errors.Join(errs...)
// }

// decodeTemplateField reads from a buffer reference to decode a field. It decodes the field's id
// first, and then looks up the FieldBuilder prototype for the field for
// further decoding the data type accordingly. It injects managers and the length decoded from
// the template. Note that for variable-length encoded fields have length of 0xFFFF set, and
// the actual length is only decoded as soon as Field.Decode() is called on VariableLengthField.
//
// decodeTemplateField is effectively only used by decoding methods for Templates and OptionsTemplates.
// Decoding data records is done in DecodeUsingTemplate with a slice of Fields.
func decodeTemplateField(r io.Reader, fieldCache FieldCache, templateCache TemplateCache) (Field, error) {
	var rawFieldId, fieldId, fieldLength uint16
	var enterpriseId uint32
	var reverse bool

	err := binary.Read(r, binary.BigEndian, &rawFieldId)
	if err != nil {
		return nil, err
	}

	penMask := uint16(0x8000)
	fieldId = (^penMask) & rawFieldId

	// length announcement via the template: this is either fixed or variable (i.e., 0xFFFF).
	// The FieldBuilder will therefore either create a fixed-length or variable-length field
	// on FieldBuilder.Complete()
	err = binary.Read(r, binary.BigEndian, &fieldLength)
	if err != nil {
		return nil, err
	}

	// private enterprise number parsing
	if rawFieldId >= 0x8000 {
		// first bit is 1, therefore this is a enterprise-specific IE
		err = binary.Read(r, binary.BigEndian, &enterpriseId)
		if err != nil {
			return nil, err
		}

		if enterpriseId == ReversePEN && Reversible(fieldId) {
			reverse = true
			// clear enterprise id, because this would obscure lookup
			enterpriseId = 0
		}
	}

	fieldBuilder, err := fieldCache.GetBuilder(context.TODO(), NewFieldKey(enterpriseId, fieldId))
	if err != nil {
		return nil, err
	}

	return fieldBuilder.
		SetLength(fieldLength).
		SetPEN(enterpriseId).
		SetReversed(reverse).
		SetFieldManager(fieldCache).
		SetTemplateManager(templateCache).
		Complete(), nil
}

// func decodeUsingTemplate(r io.Reader, fields []Field) ([]Field, error) {
// 	dfs := make([]Field, 0, len(fields))
// 	for idx, templateField := range fields {
// 		// Clone the field of the template to decode the value into while also preserving the
// 		// template information
// 		tf := templateField.Clone()
// 		name := tf.Name()
// 		n, err := tf.Decode(r)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to decode field (%d, %d/%d [%s]), %w", idx, tf.PEN(), tf.Id(), name, err)
// 		}
// 		dfs = append(dfs, tf)
// 	}
// 	return dfs, nil
// }

func (d *Decoder) initMetrics() {
	// set this so that we don't get too many empty data points in prometheus
	PacketsTotal.Add(0)
	ErrorsTotal.Add(0)
	DurationMicroseconds.Observe(0)
	for _, kind := range []string{KindDataRecord, KindTemplateRecord, KindOptionsTemplateRecord} {
		DecodedSets.WithLabelValues(kind).Add(0)
		DecodedRecords.WithLabelValues(kind).Add(0)
		DroppedRecords.WithLabelValues(kind).Add(0)
	}
}

func (d *Decoder) resetMetrics() {
	d.metrics = &decoderMetrics{
		TotalLength:    0,
		DecodedSets:    0,
		DecodedRecords: 0,
		DroppedRecords: 0,
	}
}
