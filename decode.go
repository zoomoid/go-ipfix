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
	"time"
)

// Decoder is instantiated with a fieldManager and a templateManager
// such that it can decode IPFIX packets into Records containing fields
// and additionally learn new fields and templates.
type Decoder struct {
	// fieldCache stores and manages field definitions for IEs to decode into. It is injected into the decoder at creation.
	// Particularly, fieldCache is able to learn new fields from options templates and subsequent data records.
	fieldCache FieldCache

	// templateCache stores and manages templates. It is injected into the decoder at creation
	templateCache TemplateCache

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
		fieldCache:    fields,
		templateCache: templates,
		options:       options,
		metrics:       &decoderMetrics{},
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

	if d.templateCache == nil {
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
				fieldCache:    d.fieldCache,
				templateCache: d.templateCache,
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
				d.templateCache.Add(ctx, TemplateKey{
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
				templateCache: d.templateCache,
				fieldCache:    d.fieldCache,
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
				d.templateCache.Add(ctx, TemplateKey{
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
				fieldCache:    d.fieldCache,
				templateCache: d.templateCache,
			}

			template, err := d.templateCache.Get(context.TODO(), TemplateKey{
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
