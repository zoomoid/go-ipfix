package ipfix_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zoomoid/go-ipfix"
)

func Example_collectorTCP() {
	var (
		BindAddr string = "[::]:4739"
	)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal, initiating shutdown...")
		cancel()
		<-c
		os.Exit(1)
	}()

	tcpListener := ipfix.NewTCPListener(BindAddr)
	go func() {
		log.Printf("Starting TCP listener for IPFIX messages on %s", BindAddr)
		tcpListener.Listen(ctx)
	}()

	templateCache := ipfix.NewDefaultEphemeralCache()
	fieldCache := ipfix.NewEphemeralFieldCache(templateCache)

	decoder := ipfix.NewDecoder(templateCache, fieldCache, ipfix.DecoderOptions{OmitRFC5610Records: false})

	go func() {
		for {
			select {
			case raw := <-tcpListener.Messages():
				msg, err := decoder.Decode(ctx, bytes.NewBuffer(raw))
				if err != nil {
					log.Println(fmt.Errorf("failed to decode IPFIX message: %w", err))
				}
				log.Println(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
}
