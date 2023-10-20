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

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var (
	//go:embed *.html
	fsys embed.FS
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

func run(ctx context.Context) (err error) {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/ws", handleWs)

	sErr := make(chan error)
	go func() {
		sErr <- http.ListenAndServe(":8000", nil)
	}()

	select {
	case err := <-sErr:
		return fmt.Errorf("main error: starting server: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	p, err := fsys.ReadFile("index.html")
	if err != nil {
		http.Error(w, "failed to read index page", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(p))
}

func handleWs(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "Failed to connect to socket", http.StatusBadRequest)
		return
	}

	p, err := fsys.ReadFile("message.html")
	if err != nil {
		http.Error(w, "failed to read index page", http.StatusInternalServerError)
		return
	}

	go func() {
		defer conn.Close()

		for {
			// nameOfInput:string|int
			_, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				continue
			}

			err = wsutil.WriteServerMessage(conn, op, p)
			if err != nil {
				continue
			}
		}
	}()
}
