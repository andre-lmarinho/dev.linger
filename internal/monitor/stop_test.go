package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStopProcessChecksIdentityAndSendsTerm(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	start, err := processStart(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	service := Service{PID: cmd.Process.Pid, start: start + 1}
	if err := Stop(context.Background(), service); err == nil {
		t.Fatal("accepted mismatched process identity")
	}
	service.start = start
	if err := Stop(context.Background(), service); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("process did not receive SIGTERM")
	}
}

func TestStopComposeRequestsGracefulStopForEveryContainer(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan string, 2)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	t.Setenv("DOCKER_HOST", "unix://"+socket)

	service := Service{Docker: true, Members: []Service{{ContainerID: "web"}, {ContainerID: "db"}}}
	if err := Stop(context.Background(), service); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for range 2 {
		select {
		case request := <-requests:
			got[request] = true
		default:
			t.Fatal("missing stop request")
		}
	}
	for _, want := range []string{"POST /containers/web/stop?t=-1", "POST /containers/db/stop?t=-1"} {
		if !got[want] {
			t.Fatal(fmt.Sprintf("missing request %q", want))
		}
	}
}
