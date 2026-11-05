package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Service struct {
	Name, Project, Process string
	Branch                 string
	ContainerID            string
	PID                    int
	Endpoints              []Endpoint
	Members                []Container
	Subservices            []Service
	CPU, MemoryMB          float64
	Uptime                 time.Duration
	start                  uint64
}

type Container struct {
	ID, Name  string
	Endpoints []Endpoint
}

type Endpoint struct {
	Address string
	Port    uint16
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
	owners := processOwners("/proc", os.Getuid())
	return collect(listeners, owners, docker(ctx)), nil
}

func collect(listeners []listener, owners map[string]Service, containers []container) []Service {
	services := []Service{}
	processes := map[int]int{}
	containerIDs := make(map[string]bool, len(containers))
	for _, c := range containers {
		if c.ID != "" {
			containerIDs[c.ID] = true
		}
	}
	for _, l := range listeners {
		owner, ok := owners[l.inode]
		if !ok || containerIDs[owner.ContainerID] {
			continue
		}
		i, ok := processes[owner.PID]
		if !ok {
			i = len(services)
			processes[owner.PID] = i
			services = append(services, owner)
		}
		services[i].Endpoints = addEndpoint(services[i].Endpoints, Endpoint{l.address, l.port})
	}
	groups := map[string]int{}
	for _, c := range containers {
		name := strings.TrimPrefix(first(c.Names), "/")
		if name == "" {
			name = c.ID
			if len(name) > 12 {
				name = name[:12]
			}
		}
		project := c.Labels["com.docker.compose.project"]
		key := "container:" + c.ID
		if project != "" {
			key = "compose:" + project
		}
		i, ok := groups[key]
		uptime := time.Duration(0)
		if c.Created > 0 {
			uptime = time.Since(time.Unix(c.Created, 0)).Truncate(time.Second)
			if uptime < 0 {
				uptime = 0
			}
		}
		if !ok {
			i = len(services)
			groups[key] = i
			title := name
			if project != "" {
				title = project
			}
			services = append(services, Service{Name: title, Process: "Docker", Uptime: uptime})
		} else if uptime > services[i].Uptime {
			services[i].Uptime = uptime
		}
		member := Container{ID: c.ID, Name: name}
		for _, p := range c.Ports {
			if p.Type == "tcp" && p.PublicPort != 0 {
				member.Endpoints = addEndpoint(member.Endpoints, Endpoint{p.IP, p.PublicPort})
			}
		}
		if project != "" {
			services[i].Members = append(services[i].Members, member)
		} else {
			services[i].Endpoints = member.Endpoints
			services[i].ContainerID = c.ID
		}
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].Name != services[j].Name {
			return services[i].Name < services[j].Name
		}
		return services[i].PID < services[j].PID
	})
	return services
}

func addEndpoint(endpoints []Endpoint, endpoint Endpoint) []Endpoint {
	for _, e := range endpoints {
		if e == endpoint {
			return endpoints
		}
	}
	return append(endpoints, endpoint)
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

type process struct {
	pid, parent int
	start       uint64
	cwd, name   string
	args        []string
}

func processOwners(proc string, uid int) map[string]Service {
	processes := readProcesses(proc, uid)
	return associateSockets(proc, processes)
}

func readProcesses(proc string, uid int) map[int]process {
	processes := map[int]process{}
	entries, _ := os.ReadDir(proc)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		base := filepath.Join(proc, e.Name())
		info, err := os.Stat(base)
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || int(stat.Uid) != uid {
			continue
		}
		data, err := os.ReadFile(filepath.Join(base, "stat"))
		if err != nil {
			continue
		}
		end := strings.LastIndexByte(string(data), ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(data[end+1:]))
		if len(fields) < 20 {
			continue
		}
		parent, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		start, err := strconv.ParseUint(fields[19], 10, 64)
		if err != nil {
			continue
		}
		cmd, err := os.ReadFile(filepath.Join(base, "cmdline"))
		if err != nil {
			continue
		}
		cwd, _ := os.Readlink(filepath.Join(base, "cwd"))
		name, _ := os.ReadFile(filepath.Join(base, "comm"))
		processes[pid] = process{pid: pid, parent: parent, start: start, cwd: cwd, name: strings.TrimSpace(string(name)), args: strings.Split(strings.TrimRight(string(cmd), "\x00"), "\x00")}
	}
	return processes
}

func associateSockets(proc string, processes map[int]process) map[string]Service {
	type association struct {
		pid        int
		service    Service
		recognized bool
		shared     bool
	}
	sockets := map[string]association{}
	for pid, p := range processes {
		name, evidence, ok := recognized(p, processes)
		service := Service{Name: name, PID: p.pid, Process: p.name, Project: evidence.cwd, Branch: gitBranch(evidence.cwd), start: p.start}
		base := filepath.Join(proc, strconv.Itoa(pid))
		if data, err := os.ReadFile(filepath.Join(base, "cgroup")); ok && err == nil {
			service.ContainerID = dockerCgroupID(string(data))
		}
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
			existing, exists := sockets[inode]
			if exists && existing.pid != pid {
				existing.shared = true
				sockets[inode] = existing
			} else if !exists {
				sockets[inode] = association{pid: pid, service: service, recognized: ok}
			}
		}
	}
	owners := map[string]Service{}
	for inode, socket := range sockets {
		if socket.recognized && !socket.shared {
			owners[inode] = socket.service
		}
	}
	return owners
}

func gitBranch(project string) string {
	if project == "" {
		return ""
	}
	git := filepath.Join(project, ".git")
	if data, err := os.ReadFile(git); err == nil {
		line := strings.TrimSpace(string(data))
		if strings.HasPrefix(line, "gitdir: ") {
			git = strings.TrimPrefix(line, "gitdir: ")
			if !filepath.IsAbs(git) {
				git = filepath.Join(project, git)
			}
		}
	}
	head, err := os.ReadFile(filepath.Join(git, "HEAD"))
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(head))
	if strings.HasPrefix(value, "ref: refs/heads/") {
		return strings.TrimPrefix(value, "ref: refs/heads/")
	}
	return ""
}

func dockerCgroupID(cgroup string) string {
	for _, line := range strings.Split(cgroup, "\n") {
		fields := strings.SplitN(line, ":", 3)
		if len(fields) != 3 {
			continue
		}
		parts := strings.Split(fields[2], "/")
		for i, part := range parts {
			id := ""
			if strings.HasPrefix(part, "docker-") && strings.HasSuffix(part, ".scope") {
				id = strings.TrimSuffix(strings.TrimPrefix(part, "docker-"), ".scope")
			}
			if i > 0 && parts[i-1] == "docker" {
				id = part
			}
			if len(id) != 64 {
				continue
			}
			valid := true
			for _, c := range id {
				if !strings.ContainsRune("0123456789abcdef", c) {
					valid = false
					break
				}
			}
			if valid {
				return id
			}
		}
	}
	return ""
}

func recognized(p process, processes map[int]process) (string, process, bool) {
	if name := catalog(p.args, p.cwd); name != "" {
		return name, p, true
	}
	parent, ok := processes[p.parent]
	if !ok || parent.pid == p.pid || parent.start > p.start || p.cwd == "" || p.cwd != parent.cwd {
		return "", process{}, false
	}
	if name := catalog(parent.args, parent.cwd); name != "" {
		return name, parent, true
	}
	return "", process{}, false
}

type container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Labels  map[string]string `json:"Labels"`
	Created int64             `json:"Created"`
	Ports   []struct {
		IP         string `json:"IP"`
		PublicPort uint16 `json:"PublicPort"`
		Type       string `json:"Type"`
	} `json:"Ports"`
}

func docker(ctx context.Context) []container {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		if !strings.HasPrefix(host, "unix:///") {
			return nil
		}
		return dockerAt(ctx, strings.TrimPrefix(host, "unix://"))
	}
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		path := filepath.Join(runtime, "docker.sock")
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return dockerAt(ctx, path)
		}
	}
	return dockerAt(ctx, "/var/run/docker.sock")
}

func dockerAt(ctx context.Context, socket string) []container {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
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
