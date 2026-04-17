// Package main is the entry point for the sneakystack standalone proxy.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donaldgifford/libtftest/sneakystack"
)

func main() {
	downstream := flag.String("downstream", "http://localhost:4566", "LocalStack downstream URL")
	port := flag.Int("port", 4567, "Port to listen on")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	store := sneakystack.NewMapStore()

	proxy, err := sneakystack.NewProxy(store, *downstream)
	if err != nil {
		logger.Error("create proxy", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{
		Addr:         addr,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	// Graceful shutdown on signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shutdown", "error", err)
		}
	}()

	logger.Info("sneakystack starting", "addr", addr, "downstream", *downstream)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
}
