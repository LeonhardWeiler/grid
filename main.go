package main

import (
	"context"
	"errors"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grid/internal"
)

func main() {
	mime.AddExtensionType(".js", "application/javascript")
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
	files := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /healthz", hub.Health)
	mux.HandleFunc("GET /readyz", hub.Health)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { internal.HandleWS(hub, w, r) })
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.RunAutosave(ctx)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	stopped := make(chan struct{})
	go func() {
		var sig os.Signal
		select {
		case sig = <-signals:
		case err := <-hub.Failures():
			log.Printf("storage failure; restarting with journal intact: %v", err)
		}
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
		cancel()
		if err := hub.Shutdown(); err != nil {
			log.Printf("shutdown save failed (journal retained): %v", err)
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
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
