package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
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
	// nats
	ns, err := server.NewServer(&server.Options{})
	if err != nil {
		return err
	}
	ns.Start()

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		return err
	}
	// ./nats

	http.HandleFunc("/", handleIndex())
	http.HandleFunc("/ws", handleWs(nc))

	sErr := make(chan error)
	go func() {
		sErr <- http.ListenAndServe(":8000", nil)
	}()

	select {
	case err := <-sErr:
		return fmt.Errorf("main error: starting server: %w", err)
	case <-ctx.Done():
		ns.Shutdown()
		return nil
	}
}

func handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := fsys.ReadFile("index.html")
		if err != nil {
			http.Error(w, "failed to read index page", http.StatusInternalServerError)
			return
		}
		w.Write(p)
	}
}

func handleWs(nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			http.Error(w, "Failed to connect to socket", http.StatusBadRequest)
			return
		}

		var send = make(chan []byte)

		go write(conn, send)
		go read(conn, send)
	}
}

func write(conn net.Conn, send chan []byte) {
	defer conn.Close()

	for {
		p := <-send

		err := wsutil.WriteServerText(conn, p)
		if err != nil {
			continue
		}
	}
}

func read(conn net.Conn, send chan []byte) {
	defer func() {
		close(send)
		conn.Close()
	}()

	p, _ := fsys.ReadFile("message.html")
	for {
		// nameOfInput:string|int
		_, err := wsutil.ReadClientText(conn)
		if err != nil {
			continue
		}

		send <- p
	}
}
