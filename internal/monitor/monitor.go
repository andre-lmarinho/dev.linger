package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Name, Address, Project, Process string
	Port                            uint16
	PID                             int
}

type listener struct {
	address string
	port    uint16
	inode   string
}

// Scan returns local listeners and running Docker containers. Docker is optional.
func Scan(ctx context.Context) ([]Service, error) {
	listeners, err := readListeners("/proc")
	if err != nil {
		return nil, err
	}
	owners := processOwners("/proc")
	containers := docker(ctx)
	services := make([]Service, 0, len(listeners)+len(containers))
	for _, l := range listeners {
		owner := owners[l.inode]
		if owner.Process == "docker-proxy" && published(containers, l) {
			continue
		}
		owner.Address, owner.Port = l.address, l.port
		if owner.Process == "" {
			owner.Name = "Unknown process"
		} else {
			owner.Name = owner.Process
		}
		services = append(services, owner)
	}
	for _, c := range containers {
		name := strings.TrimPrefix(first(c.Names), "/")
		if name == "" {
			name = c.ID
			if len(name) > 12 {
				name = name[:12]
			}
		}
		project := c.Labels["com.docker.compose.project"]
		if len(c.Ports) == 0 {
			services = append(services, Service{Name: name, Project: project, Process: "Docker"})
			continue
		}
		for _, p := range c.Ports {
			if p.Type != "tcp" || p.PublicPort == 0 {
				continue
			}
			services = append(services, Service{Name: name, Address: p.IP, Port: p.PublicPort, Project: project, Process: "Docker"})
		}
		// Containers without published TCP ports still belong in the menu.
		found := false
		for _, p := range c.Ports {
			if p.Type == "tcp" && p.PublicPort != 0 {
				found = true
				break
			}
		}
		if !found {
			services = append(services, Service{Name: name, Project: project, Process: "Docker"})
		}
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].Name != services[j].Name {
			return services[i].Name < services[j].Name
		}
		return services[i].Port < services[j].Port
	})
	return services, nil
}

func readListeners(proc string) ([]listener, error) {
	var result []listener
	var firstErr error
	for _, name := range []string{"tcp", "tcp6"} {
		f, err := os.Open(filepath.Join(proc, "net", name))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			fields := strings.Fields(s.Text())
			if len(fields) < 10 || fields[3] != "0A" {
				continue
			}
			parts := strings.Split(fields[1], ":")
			if len(parts) != 2 {
				continue
			}
			port, err := strconv.ParseUint(parts[1], 16, 16)
			if err != nil {
				continue
			}
			address := decodeAddress(parts[0])
			if address == "" {
				continue
			}
			result = append(result, listener{address, uint16(port), fields[9]})
		}
		if err := s.Err(); err != nil && firstErr == nil {
			firstErr = err
		}
		f.Close()
	}
	if result == nil && firstErr != nil {
		return nil, firstErr
	}
	return result, nil
}

func decodeAddress(hex string) string {
	if len(hex) != 8 && len(hex) != 32 {
		return ""
	}
	b := make([]byte, len(hex)/2)
	for i := range b {
		v, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return ""
		}
		b[i] = byte(v)
	}
	for i := 0; i < len(b); i += 4 {
		b[i], b[i+3] = b[i+3], b[i]
		b[i+1], b[i+2] = b[i+2], b[i+1]
	}
	return net.IP(b).String()
}

func processOwners(proc string) map[string]Service {
	owners := map[string]Service{}
	entries, _ := os.ReadDir(proc)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		base := filepath.Join(proc, e.Name())
		fds, err := os.ReadDir(filepath.Join(base, "fd"))
		if err != nil {
			continue
		}
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(base, "fd", fd.Name()))
			if err != nil || !strings.HasPrefix(target, "socket:[") || !strings.HasSuffix(target, "]") {
				continue
			}
			inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
			if _, exists := owners[inode]; exists {
				continue
			}
			name, _ := os.ReadFile(filepath.Join(base, "comm"))
			cwd, _ := os.Readlink(filepath.Join(base, "cwd"))
			owners[inode] = Service{PID: pid, Process: strings.TrimSpace(string(name)), Project: cwd}
		}
	}
	return owners
}

type container struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
	Ports  []struct {
		IP         string `json:"IP"`
		PublicPort uint16 `json:"PublicPort"`
		Type       string `json:"Type"`
	} `json:"Ports"`
}

func docker(ctx context.Context) []container {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://docker/containers/json", nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var result []container
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return nil
	}
	return result
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func published(containers []container, l listener) bool {
	for _, c := range containers {
		for _, p := range c.Ports {
			if p.Type == "tcp" && p.PublicPort == l.port && p.IP == l.address {
				return true
			}
		}
	}
	return false
}

func (s Service) String() string {
	parts := []string{s.Name}
	if s.Port != 0 {
		parts = append(parts, net.JoinHostPort(s.Address, strconv.Itoa(int(s.Port))))
	}
	if s.Process != "" && s.Process != s.Name {
		parts = append(parts, s.Process)
	}
	if s.PID != 0 {
		parts = append(parts, fmt.Sprintf("PID %d", s.PID))
	}
	if s.Project != "" {
		parts = append(parts, s.Project)
	}
	return strings.Join(parts, " · ")
}
