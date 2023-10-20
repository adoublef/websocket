package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
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
		sErr <- http.ListenAndServe(":8080", nil)
	}()

	select {
	case err := <-sErr:
		return fmt.Errorf("main error: starting server: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	renderHttp(w, "index.html", nil)
}

func handleWs(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "Failed to connect to socket", http.StatusBadRequest)
		return
	}
	var send = make(chan []byte)

	go write(conn, send)
	go read(conn, send)
}

func write(conn net.Conn, send chan []byte) {
	defer conn.Close()
	for {
		p, err := renderBytes("message.html", <-send)
		if err != nil {
			continue
		}
		err = wsutil.WriteServerText(conn, p)
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
	for {
		// nameOfInput:string|int
		p, err := wsutil.ReadClientText(conn)
		if err != nil {
			continue
		}

		var msg map[string]any
		_ = json.Unmarshal(p, &msg)

		send <- []byte(msg["send"].(string))
	}
}

func renderHttp(w http.ResponseWriter, name string, data any) {
	t, err := template.New(name).ParseFS(fsys, name)
	if err != nil {
		http.Error(w, "failed to parse template", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "failed to write template", http.StatusInternalServerError)
		return
	}
}

func renderBytes(name string, data any) ([]byte, error) {
	t, err := template.New(name).ParseFS(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("error writing to buffer: %w", err)
	}

	return buf.Bytes(), nil
}
