package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mittalsukhansh/forge/internal/api"
	"github.com/mittalsukhansh/forge/internal/engine"
	"github.com/mittalsukhansh/forge/internal/pipeline"
	"github.com/mittalsukhansh/forge/internal/queue"
	"github.com/mittalsukhansh/forge/internal/store"
)

func serve(ctx context.Context) error {
	path := os.Getenv("FORGE_DB")

	if path == "" {
		path = "forge.db"
	}

	s, err := store.New(path)
	if err != nil {
		return err
	}
	defer s.Close()

	workerCtx, cancel := context.WithCancel(ctx)

	q := queue.New(workerCtx, s, 2)

	defer func() {
		cancel()
		q.Wait()
	}()

	addr := os.Getenv("FORGE_ADDR")

	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.New(s, q),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdown, stop := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer stop()

		server.Shutdown(shutdown)
	}()

	log.Printf("FORGE_SERVER http://%s", addr)
	log.Println("WORKERS: 2")
	log.Println("DATABASE:", path)

	err = server.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func run(ctx context.Context) error {
	if len(os.Args) == 2 && os.Args[1] == "serve" {
		return serve(ctx)
	}

	if len(os.Args) == 3 && os.Args[1] == "run" {
		p, err := pipeline.Load(os.Args[2])
		if err != nil {
			return err
		}

		return engine.New().Run(ctx, p, os.Stdout)
	}

	return fmt.Errorf(
		"usage: forge serve | forge run <pipeline.yml>",
	)
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
