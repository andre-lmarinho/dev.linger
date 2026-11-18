package monitor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Stop asks a tracked service to terminate without escalating to a forced stop.
func Stop(ctx context.Context, service Service) error {
	if service.Docker {
		return stopContainers(ctx, service)
	}
	if service.PID <= 0 || service.start == 0 {
		return errors.New("service has no process identity")
	}
	start, err := processStart(service.PID)
	if err != nil {
		return err
	}
	if start != service.start {
		return errors.New("tracked process no longer matches PID")
	}
	return syscall.Kill(service.PID, syscall.SIGTERM)
}

func processStart(pid int) (uint64, error) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, err
	}
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return 0, errors.New("invalid process stat")
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 {
		return 0, errors.New("invalid process stat")
	}
	return strconv.ParseUint(fields[19], 10, 64)
}

func stopContainers(ctx context.Context, service Service) error {
	ids := []string{service.ContainerID}
	if len(service.Members) > 0 {
		ids = ids[:0]
		for _, member := range service.Members {
			ids = append(ids, member.ContainerID)
		}
	}
	socket, ok := dockerSocket()
	if !ok {
		return errors.New("Docker host is not a local Unix socket")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	errs := make(chan error, len(ids))
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if id == "" {
				errs <- errors.New("container has no identity")
				return
			}
			endpoint := "http://docker/containers/" + url.PathEscape(id) + "/stop?t=-1"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
			if err == nil {
				var resp *http.Response
				resp, err = client.Do(req)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
						err = fmt.Errorf("Docker stop returned %s", resp.Status)
					}
				}
			}
			if err != nil {
				errs <- fmt.Errorf("stop container %s: %w", id, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	var result []error
	for err := range errs {
		result = append(result, err)
	}
	return errors.Join(result...)
}
