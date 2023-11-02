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
// The decoder is fed with messages from the reader the same way we previously did in
// the TCP and UDP listener example, using a indefinitely running goroutine with message channels.
func Example_decoder() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, _ := os.Open("demo_flow_records.ipfix")
	defer f.Close()

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
