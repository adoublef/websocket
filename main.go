package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	//go:embed index.html
	index embed.FS

	//go:embed message.html
	message embed.FS
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
	}
}

// sudo lsof -t -i tcp:8000 | xargs kill -9
func run(ctx context.Context) (err error) {
	sErr := make(chan error)
	go func() {
		sErr <- http.ListenAndServe(":8000", http.HandlerFunc(handleIndex))
	}()

	select {
	case err := <-sErr:
		return fmt.Errorf("main error: starting server: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	p, err := index.ReadFile("index.html")
	if err != nil {
		http.Error(w, "failed to read index page", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(p))
}

func handleWs(w http.ResponseWriter, r *http.Request) {

}
