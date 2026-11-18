package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The workload shape is the contract documented in benchmarks/README.md;
// these numbers and that table change together.
const (
	controlledProcesses = 200
	controlledListeners = 20
	recognizedProcs     = 3  // processes owning the recognized listeners
	recognizedListeners = 10 // listeners shared among recognizedProcs
	otherListeners      = 5  // listeners owned by unrecognized processes
	sharedListeners     = 1  // one inode held by two processes; must be omitted
	unownedListeners    = 4  // no process links them
)

// sharedOwnerIndexes are fixture process indexes of the two processes that
// hold the shared listener between them.
var sharedOwnerIndexes = []int{20, 21}

func fixturePID(i int) int { return 1000 + i }

// controlledProc creates a fixed workload without inspecting the host.
func controlledProc(tb testing.TB) string {
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
	for i := 0; i < controlledListeners; i++ {
		fmt.Fprintf(&tcp, "%d: 0100007F:%04X 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 %d\n", i, 3000+i, 5000+i)
	}
	for i := 0; i < controlledProcesses; i++ {
		pid := 1000 + i
		base := filepath.Join(root, fmt.Sprint(pid))
		if err := os.MkdirAll(filepath.Join(base, "fd"), 0755); err != nil {
			tb.Fatal(err)
		}
		write(filepath.Join(base, "stat"), fmt.Sprintf("%d (proc) S 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 %d 0", pid, pid))
		cmd := "unrecognized\x00"
		if i < recognizedProcs || slices.Contains(sharedOwnerIndexes, i) {
			cmd = "python3\x00-m\x00http.server\x00"
		}
		write(filepath.Join(base, "cmdline"), cmd)
		link(root, filepath.Join(base, "cwd"))
		for fd := 3; fd < 10; fd++ {
			link("/dev/null", filepath.Join(base, "fd", fmt.Sprint(fd)))
		}
	}
	for i := 0; i < recognizedListeners; i++ {
		pid := fixturePID(i % recognizedProcs)
		link(fmt.Sprintf("socket:[%d]", 5000+i), filepath.Join(root, fmt.Sprint(pid), "fd", fmt.Sprint(20+i)))
	}
	for i := recognizedListeners; i < recognizedListeners+otherListeners; i++ {
		pid := fixturePID(i)
		link(fmt.Sprintf("socket:[%d]", 5000+i), filepath.Join(root, fmt.Sprint(pid), "fd", "20"))
	}
	sharedInode := 5000 + recognizedListeners + otherListeners
	for _, index := range sharedOwnerIndexes {
		link(fmt.Sprintf("socket:[%d]", sharedInode), filepath.Join(root, fmt.Sprint(fixturePID(index)), "fd", "20"))
	}
	write(filepath.Join(root, "net", "tcp"), tcp.String())
	write(filepath.Join(root, "net", "tcp6"), "header\n")
	return root
}

func controlledContainers() []container {
	var containers []container
	for _, project := range []string{"alpha", "beta"} {
		for i := 1; i <= 3; i++ {
			containers = append(containers, container{
				ID:     fmt.Sprintf("%s-%d", project, i),
				Names:  []string{fmt.Sprintf("/%s-%d", project, i)},
				Labels: map[string]string{"com.docker.compose.project": project},
			})
		}
	}
	return append(containers,
		container{ID: "standalone-a", Names: []string{"/standalone-a"}, Labels: map[string]string{}},
		container{ID: "standalone-b", Names: []string{"/standalone-b"}, Labels: map[string]string{}},
	)
}

func TestControlledWorkload(t *testing.T) {
	root := controlledProc(t)
	if processes := readProcesses(root, os.Getuid()); len(processes) != controlledProcesses {
		t.Fatalf("processes: got %d, want %d", len(processes), controlledProcesses)
	}
	listeners, err := readListeners(root)
	if err != nil || len(listeners) != controlledListeners {
		t.Fatalf("listeners: %d, error: %v", len(listeners), err)
	}
	owners := processOwners(root, os.Getuid(), listeners)
	if len(owners) != recognizedListeners || owners["5000"].Name != "Python HTTP server" {
		t.Fatalf("unexpected owners: %#v", owners)
	}
	if _, ok := owners["5015"]; ok {
		t.Fatal("shared listener was attributed to one process")
	}
	containers := controlledContainers()
	if len(containers) != 8 {
		t.Fatalf("containers: got %d, want 8", len(containers))
	}
	services := collect(listeners, owners, containers)
	if len(services) != 7 {
		t.Fatalf("got %d services, want 7", len(services))
	}
	var local, compose, standalone int
	for _, service := range services {
		switch {
		case !service.Docker:
			local++
		case len(service.Members) == 3:
			compose++
		case service.ContainerID != "":
			standalone++
		}
	}
	if local != 3 || compose != 2 || standalone != 2 {
		t.Fatalf("services: local=%d compose=%d standalone=%d", local, compose, standalone)
	}
}

func TestListenerIndexPreservesSharedOwnership(t *testing.T) {
	root := controlledProc(t)
	id := strings.Repeat("a", 64)
	if err := os.WriteFile(filepath.Join(root, "1000", "cgroup"), []byte("0::/docker/"+id+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	listeners := []listener{{inode: "5000"}, {inode: "5015"}}
	owners := processOwners(root, os.Getuid(), listeners)
	if len(owners) != 1 || owners["5000"].PID != 1000 || owners["5000"].ContainerID != id {
		t.Fatalf("unexpected owners: %#v", owners)
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

func BenchmarkControlledWorkload(b *testing.B) {
	root := controlledProc(b)
	containers := controlledContainers()
	processes := readProcesses(root, os.Getuid())
	listeners, err := readListeners(root)
	if err != nil {
		b.Fatal(err)
	}
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
		{"AssociateSockets", func() { associateSockets(root, processes, listeners) }},
		{"Discovery", func() {
			listeners, err := readListeners(root)
			if err != nil {
				b.Fatal(err)
			}
			collect(listeners, processOwners(root, os.Getuid(), listeners), containers)
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
