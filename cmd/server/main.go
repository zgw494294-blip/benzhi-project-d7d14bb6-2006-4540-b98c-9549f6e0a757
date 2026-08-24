package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/httpapi"
	"timber-release-gate/internal/store"
)

func main() {
	addrFlag := flag.String("addr", "", "监听地址")
	self := flag.Bool("selfcheck", false, "执行完整自检")
	flag.Parse()

	addr, err := resolveAddr(*addrFlag, os.Getenv("PORT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	dataDir := os.Getenv("TIMBER_DATA_DIR")
	if *self {
		dataDir, err = os.MkdirTemp("", "timber-selfcheck-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer os.RemoveAll(dataDir)
	}
	st, err := store.Open(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	svc := application.New(st)
	api := httpapi.New(svc)
	server := &http.Server{
		Addr:              addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      8 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	if *self {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := runSelfcheck(ctx, addr, server); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("selfcheck passed")
		return
	}
	if err := serve(addr, server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(addr string, server *http.Server) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听失败: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	fmt.Printf("古建木构修缮放行台监听 %s\n", addr)
	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
