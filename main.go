package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/devlinger/devlinger/internal/app"
	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/devlinger/devlinger/internal/tray"
)

// scanInterval is how often the tray refreshes its service list; the README
// documents the same number to users.
const scanInterval = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	instance, acquired, err := app.Acquire()
	if err != nil {
		log.Fatal(err)
	}
	if !acquired {
		return
	}
	defer instance.Close()

	t, err := tray.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()
	go func() {
		ticker := time.NewTicker(scanInterval)
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
	<-ctx.Done()
}
