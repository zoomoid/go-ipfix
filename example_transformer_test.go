package ipfix_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zoomoid/go-ipfix"
)

// Transforms IPFIX messages containing more than one record and template set per message into
// a stream of messages that only contain one record in one typed set per message.
// Note that while this obviously includes a lot of overhead, it is helpful in scenarios where
// we want _individual records_ to end up in a hypothetical database, because queries and filters
// are much easier implemented, and the grouping of records in homogeneous sets is mostly only
// done for reducing message overhead from redundancy.
// In practice, due to the possibly complex timing of flow records, sets often only contain single
// records anyways and rarely do messages contain more than one set.
// Software exporters such as yaf behave differently in terms of message packing when writing
// to files or sending via TCP or UDP. To ease this difference, normalization appears reasonable.
func Example_transformerNormalizeRecords() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in, _ := os.Open("demo_flow_records.ipfix")
	defer in.Close()

	out, _ := os.CreateTemp("", "normalized_flow_records.ipfix")

	r := ipfix.NewIPFIXFileReader(in)
	go r.Start(ctx)

	templateCache := ipfix.NewDefaultEphemeralCache()
	fieldCache := ipfix.NewEphemeralFieldCache(templateCache)

	decoder := ipfix.NewDecoder(templateCache, fieldCache, ipfix.DecoderOptions{OmitRFC5610Records: false})

	go func() {
		for {
			select {
			case raw := <-r.Messages():
				msg, err := decoder.Decode(ctx, bytes.NewBuffer(raw))
				if err != nil {
					log.Fatalln(fmt.Errorf("failed to decode IPFIX message: %w", err))
				}
				normalizedMessages, err := NormalizeIPFIXMessage(msg)
				if err != nil {
					log.Fatalln(fmt.Errorf("failed to normalize IPFIX message: %w", err))
				}
				for _, newMsg := range normalizedMessages {
					_, err := newMsg.Encode(out)
					if err != nil {
						log.Fatalln(fmt.Errorf("failed to encode IPFIX message: %w", err))
					}
				}
			case err := <-r.Errors():
				log.Println(fmt.Errorf("failed to read IPFIX message: %w", err))
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
}

const (
	ipfixPacketHeaderLength int = 16
	ipfixSetHeaderLength    int = 4
)

var (
	sequenceNumber uint32 = 0
)

func NormalizeIPFIXMessage(old *ipfix.Message) (new []*ipfix.Message, err error) {
	new = make([]*ipfix.Message, 0)
	for _, fs := range old.Sets {
		switch fss := fs.Set.(type) {
		case *ipfix.TemplateSet:
			for _, rr := range fss.Records {
				flow := &bytes.Buffer{}
				n, err := rr.Encode(flow) // we use this to determine the NEW set length!
				if err != nil {
					return nil, err // skip entire packet
				}
				pp := &ipfix.Message{
					Version:             10,
					ExportTime:          old.ExportTime,
					SequenceNumber:      uint32(sequenceNumber), // this needs to be rewritten!
					ObservationDomainId: old.ObservationDomainId,
					Length:              uint16(n + ipfixPacketHeaderLength + ipfixSetHeaderLength),
					Sets: []ipfix.Set{
						{
							SetHeader: ipfix.SetHeader{
								Id:     fs.Id,
								Length: uint16(n + ipfixSetHeaderLength), // single record length + set header length
							},
							Set: &ipfix.TemplateSet{
								Records: []ipfix.TemplateRecord{rr},
							},
						},
					},
				}
				// sequenceNumber++ - RFC 7011: "Template and Options Template Records do not increase the Sequence Number."
				new = append(new, pp)
			}
		case *ipfix.OptionsTemplateSet:
			for _, rr := range fss.Records {
				flow := &bytes.Buffer{}
				n, err := rr.Encode(flow) // we use this to determine the NEW set length!
				if err != nil {
					return nil, err // skip entire packet
				}
				pp := &ipfix.Message{
					Version:             10,
					ExportTime:          old.ExportTime,
					SequenceNumber:      uint32(sequenceNumber), // this needs to be rewritten!
					ObservationDomainId: old.ObservationDomainId,
					Length:              uint16(n + ipfixPacketHeaderLength + ipfixSetHeaderLength),
					Sets: []ipfix.Set{
						{
							SetHeader: ipfix.SetHeader{
								Id:     fs.Id,
								Length: uint16(n + ipfixSetHeaderLength), // single record length + set header length
							},
							Set: &ipfix.OptionsTemplateSet{
								Records: []ipfix.OptionsTemplateRecord{rr},
							},
						},
					},
				}
				// sequenceNumber++ - RFC 7011: "Template and Options Template Records do not increase the Sequence Number."
				new = append(new, pp)
				// recordCounter++
			}
		case *ipfix.DataSet:
			for _, rr := range fss.Records {
				flow := &bytes.Buffer{}
				n, err := rr.Encode(flow) // we use this to determine the *new* set length!
				if err != nil {
					return nil, err // skip entire packet
				}
				pp := &ipfix.Message{
					Version:             10,
					ExportTime:          old.ExportTime,
					SequenceNumber:      uint32(sequenceNumber), // this needs to be rewritten!
					ObservationDomainId: old.ObservationDomainId,
					Length:              uint16(n + ipfixPacketHeaderLength + ipfixSetHeaderLength),
					Sets: []ipfix.Set{
						{
							SetHeader: ipfix.SetHeader{
								Id:     fs.Id,
								Length: uint16(n + ipfixSetHeaderLength), // single record length + set header length
							},
							Set: &ipfix.DataSet{
								Records: []ipfix.DataRecord{rr},
							},
						},
					},
				}
				sequenceNumber++
				new = append(new, pp)
			}
		}
	}
	return

}
