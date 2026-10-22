package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grid/internal"
)

func main() {
	probe := flag.String("probe", "", "check HTTP readiness and a read-only WebSocket snapshot")
	check := flag.String("check-state", "", "validate a restored state directory")
	version := flag.Bool("version", false, "print the build revision")
	flag.Parse()
	if *version {
		fmt.Println(internal.Revision)
		return
	}
	if *check != "" {
		if err := internal.CheckState(*check); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *probe != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := internal.Probe(ctx, *probe); err != nil {
			log.Fatal(err)
		}
		fmt.Println("ready: HTTP and WebSocket snapshot")
		return
	}
	mime.AddExtensionType(".js", "application/javascript")
	dir := os.Getenv("STATE_DIRECTORY")
	if dir == "" {
		dir = "./data"
	}
	lock, err := internal.LockStateDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Close()
	hub := internal.NewHub()
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./public/dist"
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:4000"
	}
	mux := http.NewServeMux()
	mux.Handle("/", internal.StaticHandler(staticDir))
	mux.HandleFunc("GET /healthz", hub.Health)
	mux.HandleFunc("GET /readyz", hub.Health)
	mux.HandleFunc("GET /version", internal.Version)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { internal.HandleWS(hub, w, r) })
	server := &http.Server{Addr: addr, Handler: internal.SecurityHeaders(mux), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.RunAutosave(ctx)
	go hub.RunStatusLog(ctx)
	var admin *http.Server
	if adminAddr := os.Getenv("ADMIN_ADDR"); adminAddr != "" {
		host, _, err := net.SplitHostPort(adminAddr)
		if err != nil || !net.ParseIP(host).IsLoopback() {
			log.Fatal("ADMIN_ADDR must use a literal loopback address")
		}
		listener, err := net.Listen("tcp", adminAddr)
		if err != nil {
			log.Fatal(err)
		}
		admin = &http.Server{Handler: hub.AdminHandler(), ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
		go func() {
			if err := admin.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
	}
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	go func() {
		select {
		case err := <-hub.Failures():
			log.Printf("storage failure; restarting with journal intact: %v", err)
			time.AfterFunc(10*time.Second, func() { os.Exit(1) })
			select {
			case signals <- syscall.SIGTERM:
			default:
			}
		case <-ctx.Done():
		}
	}()
	stopped := make(chan struct{})
	go func() {
		sig := <-signals
		if sig == syscall.SIGHUP {
			log.Println("reload requested; waiting for 30 seconds without drawing (at most 5 minutes)")
			quietCtx, quietCancel := context.WithCancel(ctx)
			go func() {
				for {
					select {
					case sig := <-signals:
						if sig != syscall.SIGHUP {
							quietCancel()
							return
						}
					case <-quietCtx.Done():
						return
					}
				}
			}()
			hub.WaitForQuiet(quietCtx, 30*time.Second, 5*time.Minute)
			quietCancel()
		}
		guard := time.AfterFunc(12*time.Second, func() { log.Print("shutdown timed out; journal retained"); os.Exit(1) })
		defer guard.Stop()
		cancel()
		if err := hub.Shutdown(); err != nil {
			log.Printf("shutdown save failed (journal retained): %v", err)
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
		}
		if admin != nil {
			admin.Shutdown(shutdownCtx)
		}
		close(stopped)
	}()
	log.Printf("Server running on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-stopped
	if hub.Failed() {
		os.Exit(1)
	}
}
