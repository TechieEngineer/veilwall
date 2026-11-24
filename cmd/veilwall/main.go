package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/veilwall/veilwall/internal/api"
	"github.com/veilwall/veilwall/internal/database"
	"github.com/veilwall/veilwall/internal/firewall"
	"github.com/veilwall/veilwall/web"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	port := flag.Int("port", 8080, "HTTP port")
	bind := flag.String("bind", "127.0.0.1", "loopback address only")
	demo := flag.Bool("demo", true, "simulate rules; false enables read-only Linux validation")
	path := flag.String("db", ":memory:", "SQLite file path, or :memory:")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		return errors.New("port outside range")
	}
	ip := net.ParseIP(*bind)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("v1 beta only permits loopback IP addresses")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var engine firewall.Engine = firewall.NewMockEngine()
	if !*demo {
		var e error
		engine, e = firewall.NewLinuxEngine(ctx)
		if e != nil {
			return e
		}
	}
	if *path != ":memory:" {
		if strings.HasPrefix(*path, "file:") || strings.ContainsAny(*path, "?#") {
			return errors.New("use a regular SQLite path, not a URI")
		}
		f, e := os.OpenFile(*path, os.O_CREATE|os.O_RDWR, 0600)
		if e != nil {
			return e
		}
		f.Close()
	}
	db, e := database.Open(ctx, *path)
	if e != nil {
		return e
	}
	defer db.Close()
	rules, e := db.Rules(ctx)
	if e != nil {
		return e
	}
	if e = engine.Apply(ctx, rules); e != nil {
		return e
	}
	m := firewall.NewManager(engine, db, 30*time.Second)
	defer func() {
		if e := m.Shutdown(); e != nil {
			log.Printf("rollback: %v", e)
		}
	}()
	address := net.JoinHostPort(*bind, strconv.Itoa(*port))
	handler, e := api.New(m, db, engine.Name(), address, web.Handler())
	if e != nil {
		return e
	}
	listener, e := net.Listen("tcp", address)
	if e != nil {
		return e
	}
	defer listener.Close()
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	ch := make(chan error, 1)
	go func() { ch <- srv.Serve(listener) }()
	log.Printf("VeilWall v1.0.0-beta.1 | %s | NO packet enforcement", engine.Name())
	log.Printf("Dashboard: http://%s", address)
	select {
	case <-ctx.Done():
	case e := <-ch:
		if !errors.Is(e, http.ErrServerClosed) {
			return fmt.Errorf("HTTP: %w", e)
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = srv.Shutdown(shutdown); e != nil {
		_ = srv.Close()
		return e
	}
	return m.Shutdown()
}
