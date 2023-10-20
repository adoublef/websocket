package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	q := make(chan os.Signal, 1)
	signal.Notify(q, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-q
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Printf("adoublef/websocket: %s", err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func run(ctx context.Context) (err error) {
	log.Printf("hello, %s\n", "world")
	return
}
