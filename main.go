package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/devlinger/devlinger/internal/tray"
)

func main() {
	t, err := tray.New()
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			services, err := monitor.Scan(ctx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Print(err)
			}
			t.Update(services, err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	select {
	case <-ctx.Done():
	case <-t.Done():
	}
}
