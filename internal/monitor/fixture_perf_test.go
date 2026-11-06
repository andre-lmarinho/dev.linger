package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// syntheticProc creates a stable input shape without inspecting the host's processes.
func syntheticProc(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	write := func(path, value string) {
		tb.Helper()
		if err := os.WriteFile(path, []byte(value), 0644); err != nil {
			tb.Fatal(err)
		}
	}
	link := func(target, path string) {
		tb.Helper()
		if err := os.Symlink(target, path); err != nil {
			tb.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "net"), 0755); err != nil {
		tb.Fatal(err)
	}
	var tcp strings.Builder
	tcp.WriteString("header\n")
	for i := 0; i < 100; i++ {
		pid := 1000 + i
		base := filepath.Join(root, fmt.Sprint(pid))
		if err := os.MkdirAll(filepath.Join(base, "fd"), 0755); err != nil {
			tb.Fatal(err)
		}
		write(filepath.Join(base, "stat"), fmt.Sprintf("%d (proc) S 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 %d 0", pid, pid))
		cmd := "unrecognized\x00"
		if i < 10 {
			cmd = "python3\x00-m\x00http.server\x00"
			fmt.Fprintf(&tcp, "%d: 0100007F:%04X 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 %d\n", i, 3000+i, pid)
		}
		write(filepath.Join(base, "cmdline"), cmd)
		link(root, filepath.Join(base, "cwd"))
		link(fmt.Sprintf("socket:[%d]", pid), filepath.Join(base, "fd", "3"))
		for fd := 4; fd < 10; fd++ {
			link("/dev/null", filepath.Join(base, "fd", fmt.Sprint(fd)))
		}
		if i < 2 {
			link("socket:[999999]", filepath.Join(base, "fd", "10"))
		}
	}
	write(filepath.Join(root, "net", "tcp"), tcp.String())
	write(filepath.Join(root, "net", "tcp6"), "header\n")
	return root
}

func TestSyntheticProc(t *testing.T) {
	root := syntheticProc(t)
	listeners, err := readListeners(root)
	if err != nil || len(listeners) != 10 {
		t.Fatalf("listeners: %d, error: %v", len(listeners), err)
	}
	owners := processOwners(root, os.Getuid())
	if len(owners) != 10 || owners["1000"].Name != "Python HTTP server" {
		t.Fatalf("unexpected owners: %#v", owners)
	}
	if _, ok := owners["999999"]; ok {
		t.Fatal("shared socket must not have an owner")
	}
	if services := collect(listeners, owners, nil); len(services) != 10 {
		t.Fatalf("got %d services, want 10", len(services))
	}
}

func TestCatalogDistinctCandidates(t *testing.T) {
	entries := []signature{
		{entrypoint: "/not-a-match", executables: []string{"other"}},
		{entrypoint: "/node_modules/vite/bin/vite.js", name: "Vite", executables: []string{"node"}, commands: []string{"dev"}},
	}
	if got := matchCatalog([]string{"node", "node_modules/vite/bin/vite.js", "dev"}, t.TempDir(), entries); got != "Vite" {
		t.Fatalf("got %q, want Vite", got)
	}
}

func BenchmarkSyntheticProc(b *testing.B) {
	root := syntheticProc(b)
	processes := readProcesses(root, os.Getuid())
	for _, tc := range []struct {
		name string
		run  func()
	}{
		{"Listeners", func() {
			if _, err := readListeners(root); err != nil {
				b.Fatal(err)
			}
		}},
		{"Processes", func() { readProcesses(root, os.Getuid()) }},
		{"AssociateSockets", func() { associateSockets(root, processes) }},
		{"Discovery", func() {
			listeners, err := readListeners(root)
			if err != nil {
				b.Fatal(err)
			}
			collect(listeners, processOwners(root, os.Getuid()), nil)
		}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tc.run()
			}
		})
	}
}
