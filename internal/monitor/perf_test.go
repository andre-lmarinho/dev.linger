package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// These benchmarks read the current host's /proc and optional Docker socket.
func BenchmarkLive(b *testing.B) {
	ctx := context.Background()
	services, err := Scan(ctx)
	if err != nil {
		b.Fatal(err)
	}
	processes := readProcesses("/proc", os.Getuid())
	listeners, err := readListeners("/proc")
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("fixture: %d user processes, %d listeners, %d services", len(processes), len(listeners), len(services))
	benchmarks := []struct {
		name string
		run  func()
	}{
		{"Scan", func() {
			if _, err := Scan(ctx); err != nil {
				b.Fatal(err)
			}
		}},
		{"Listeners", func() {
			if _, err := readListeners("/proc"); err != nil {
				b.Fatal(err)
			}
		}},
		{"Processes", func() { readProcesses("/proc", os.Getuid()) }},
		{"AssociateSockets", func() { associateSockets("/proc", processes, listeners) }},
		{"Docker", func() { docker(ctx) }},
	}
	var metrics Metrics
	metrics.Update(services)
	benchmarks = append(benchmarks, struct {
		name string
		run  func()
	}{"Metrics", func() { metrics.Update(services) }})
	self, ok := processes[os.Getpid()]
	if !ok {
		b.Fatal("benchmark process missing from /proc")
	}
	selfService := []Service{{PID: self.pid, start: self.start}}
	var selfMetrics Metrics
	selfMetrics.Update(selfService)
	if !selfService[0].MemoryValid {
		b.Fatal("benchmark process metrics unavailable")
	}
	benchmarks = append(benchmarks, struct {
		name string
		run  func()
	}{"MetricsSelf", func() { selfMetrics.Update(selfService) }})
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmark.run()
			}
		})
	}
}

func BenchmarkCatalog(b *testing.B) {
	dir := b.TempDir()
	entry := filepath.Join(dir, "node_modules", "vite", "bin", "vite.js")
	if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(entry, nil, 0644); err != nil {
		b.Fatal(err)
	}
	link := filepath.Join(dir, "vite")
	if err := os.Symlink(entry, link); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"symlink", []string{"node", link, "dev"}, "Vite"},
		{"unrecognized", []string{"node", entry, "build"}, ""},
	} {
		b.Run(tc.name, func(b *testing.B) {
			if got := catalog(tc.args, dir); got != tc.want {
				b.Fatalf("got %q, want %q", got, tc.want)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				catalog(tc.args, dir)
			}
		})
	}
}
