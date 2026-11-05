package monitor

import (
	"fmt"
	"math"
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
	write("42/statm", "1000 512 0 0 0 0 0\n")
	writeStat(1000, 10, 123)
	services := []Service{{PID: 42, start: 123}}
	var m Metrics
	m.update(proc, services)
	wantMemory := float64(512*os.Getpagesize()) / 1_000_000
	if services[0].CPUValid || !services[0].MemoryValid || services[0].MemoryMB != wantMemory {
		t.Fatalf("first sample: %+v", services[0])
	}
	writeStat(1200, 30, 123)
	m.update(proc, services)
	if !services[0].CPUValid || math.Abs(services[0].CPU-20) > 0.001 {
		t.Fatalf("second sample: %+v", services[0])
	}
	writeStat(1300, 30, 123)
	write("42/statm", "1000 0\n")
	m.update(proc, services)
	if !services[0].CPUValid || services[0].CPU != 0 || !services[0].MemoryValid || services[0].MemoryMB != 0 {
		t.Fatalf("valid zero sample: %+v", services[0])
	}
	write("stat", "cpu 1350 0 0 0 0 0 0 0\ncpu0 1 0 0 0\n")
	m.update(proc, services)
	if services[0].CPUValid {
		t.Fatal("CPU usage reported after CPU count changed")
	}
	writeStat(1400, 40, 124)
	m.update(proc, services)
	if services[0].CPUValid || services[0].MemoryValid {
		t.Fatalf("reused pid: %+v", services[0])
	}
	services[0].start = 124
	m.update(proc, services)
	if services[0].CPUValid {
		t.Fatal("new process used old CPU sample")
	}
}

func TestDockerHasNoInventedMetrics(t *testing.T) {
	proc := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(proc, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("meminfo", "MemTotal: 1000000 kB\nMemAvailable: 250000 kB\n")
	write("stat", "cpu 100 0 0 900 0 0 0 0\ncpu0 100 0 0 900\n")
	services := []Service{{Name: "demo", Docker: true, CPUValid: true, MemoryValid: true, Members: []Service{{Name: "web"}, {Name: "db"}}}, {Name: "standalone", Docker: true}}
	var m Metrics
	m.update(proc, services)
	write("stat", "cpu 150 0 0 950 0 0 0 0\ncpu0 150 0 0 950\n")
	m.update(proc, services)
	for _, service := range services {
		if service.CPUValid || service.MemoryValid {
			t.Fatalf("invented metrics for %s: %+v", service.Name, service)
		}
	}
}
