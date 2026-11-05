package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMetrics(t *testing.T) {
	proc := t.TempDir()
	if err := os.Mkdir(filepath.Join(proc, "42"), 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(proc, path), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeStat := func(total, ticks, start int) {
		t.Helper()
		write("stat", fmt.Sprintf("cpu  %d 0 0 0 0 0 0 0\ncpu0 1 0 0 0\ncpu1 1 0 0 0\n", total))
		write("42/stat", fmt.Sprintf("42 (a process) S 1 1 1 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0 %d 0", ticks, start))
	}
	write("42/status", "Name: test\nVmRSS: 2048 kB\n")
	writeStat(1000, 10, 123)
	services := []Service{{PID: 42, start: 123}}
	var m Metrics
	m.update(proc, services)
	if services[0].CPU == nil || *services[0].CPU != 0 || services[0].MemoryMB == nil || *services[0].MemoryMB != 2.097152 {
		t.Fatalf("first sample: %+v", services[0])
	}
	writeStat(1200, 30, 123)
	m.update(proc, services)
	if services[0].CPU == nil || *services[0].CPU != 20 {
		t.Fatalf("second sample: %+v", services[0])
	}
	writeStat(1400, 40, 124)
	m.update(proc, services)
	if services[0].CPU == nil || *services[0].CPU != 0 || services[0].MemoryMB == nil || *services[0].MemoryMB != 0 {
		t.Fatalf("reused pid: %+v", services[0])
	}
	services[0].start = 124
	m.update(proc, services)
	if services[0].CPU == nil || *services[0].CPU != 0 {
		t.Fatal("new process used old CPU sample")
	}
}

func TestDockerShowsHostMetrics(t *testing.T) {
	proc := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(proc, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("meminfo", "MemTotal: 1000000 kB\nMemAvailable: 250000 kB\n")
	write("stat", "cpu 100 0 0 900 0 0 0 0\ncpu0 100 0 0 900\n")
	services := []Service{{Name: "demo", Process: "Docker", Members: []Container{{Name: "web"}, {Name: "db"}}}, {Name: "standalone", Process: "Docker"}}
	var m Metrics
	m.update(proc, services)
	write("stat", "cpu 150 0 0 950 0 0 0 0\ncpu0 150 0 0 950\n")
	m.update(proc, services)
	for _, service := range services {
		if service.CPU == nil || *service.CPU != 50 || service.MemoryMB == nil || *service.MemoryMB != 768 {
			t.Fatalf("host metrics for %s: %+v", service.Name, service)
		}
	}
}
