package ipfix_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zoomoid/go-ipfix"
)

// A simple decoder of IPFIX messages read from a file. The example uses the
// IPFIXFileReader, which asserts the file must contain IPFIX messages according
// to RFC 5655.
func Example_ipfixFileReader() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, _ := os.Open("demo_flow_records.ipfix")
	r := ipfix.NewIPFIXFileReader(f)
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
					log.Println(fmt.Errorf("failed to decode IPFIX message: %w", err))
				}
				log.Println(msg)
			case err := <-r.Errors():
				log.Println(fmt.Errorf("failed to read IPFIX message: %w", err))
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
}

func Example_readFull() {
	f, _ := os.Open("demo_flow_records.ipfix")

	messages, err := ipfix.ReadFull(f)
	if err != nil {
		log.Fatalln(err)
	}

	templateCache := ipfix.NewDefaultEphemeralCache()
	fieldCache := ipfix.NewEphemeralFieldCache(templateCache)

	decoder := ipfix.NewDecoder(templateCache, fieldCache, ipfix.DecoderOptions{OmitRFC5610Records: false})
	for _, rawMessage := range messages {
		msg, err := decoder.Decode(context.TODO(), bytes.NewBuffer(rawMessage))
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(msg)
	}
}
