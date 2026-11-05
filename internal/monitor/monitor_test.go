package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadListeners(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "net"), 0755); err != nil {
		t.Fatal(err)
	}
	data := "  sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n  0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 12345\n  1: 0100007F:0BB9 00000000:0000 01 00000000:00000000 00:00000000 00000000 1000 0 12346\n"
	if err := os.WriteFile(filepath.Join(dir, "net", "tcp"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readListeners(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].address != "127.0.0.1" || got[0].port != 3000 || got[0].inode != "12345" {
		t.Fatalf("unexpected listeners: %#v", got)
	}
}

func TestDecodeIPv6(t *testing.T) {
	if got := decodeAddress("00000000000000000000000001000000"); got != "::1" {
		t.Fatal(got)
	}
}

func TestCatalog(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"node", "/project/node_modules/next/dist/bin/next", "dev"}, "Next.js"},
		{[]string{"node", "/project/node_modules/astro/astro.js", "dev"}, "Astro"},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js"}, "Vite"},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js", "serve"}, "Vite"},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js", "--host"}, "Vite"},
		{[]string{"node", "/project/node_modules/astro/bin/astro.mjs", "dev"}, "Astro"},
		{[]string{"node", "/project/node_modules/astro/astro.js", "--help"}, ""},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js", "build"}, ""},
		{[]string{"node", "/project/node_modules/next/dist/bin/next", "start"}, ""},
		{[]string{"opencode-cli", "serve", "--service"}, ""},
		{[]string{"code", "--type=utility"}, ""},
		{[]string{"other", "/project/node_modules/next/dist/bin/next", "dev"}, ""},
		{[]string{"node", "/tmp/next", "dev"}, ""},
	}
	for _, tt := range tests {
		if got := catalog(tt.args, "/project"); got != tt.want {
			t.Errorf("%v: got %q want %q", tt.args, got, tt.want)
		}
	}
}

func TestExpandedCatalog(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"node", "/project/node_modules/nuxt/bin/nuxt.mjs", "dev"}, "Nuxt"},
		{[]string{"node", "/project/node_modules/nuxi/bin/nuxi.mjs", "dev"}, "Nuxt"},
		{[]string{"node", "/project/node_modules/@nuxt/cli/bin/nuxi.mjs", "dev"}, "Nuxt"},
		{[]string{"node", "/project/node_modules/@angular/cli/bin/ng.js", "serve"}, "Angular"},
		{[]string{"node", "/project/node_modules/@nestjs/cli/bin/nest.js", "start"}, "NestJS"},
		{[]string{"node", "/project/node_modules/@remix-run/dev/dist/cli.js", "dev"}, "Remix"},
		{[]string{"node", "/project/node_modules/storybook/dist/bin/dispatcher.js", "dev"}, "Storybook"},
		{[]string{"node", "/project/node_modules/@storybook/cli/dist/bin/index.js", "dev"}, "Storybook"},
		{[]string{"node", "/project/node_modules/expo/bin/cli", "start"}, "Expo"},
		{[]string{"python3", "-m", "http.server", "8000"}, "Python HTTP server"},
		{[]string{"python3", "-m", "django", "runserver"}, "Django"},
		{[]string{"python", "manage.py", "runserver"}, "Django"},
		{[]string{"python3", "-m", "flask", "run"}, "Flask"},
		{[]string{"python3.12", "-m", "flask", "run", "--debug"}, "Flask"},
		{[]string{"python3.12", "-m", "http.server", "8000"}, "Python HTTP server"},
		{[]string{"/usr/bin/python3.13", "-m", "http.server", "8000"}, "Python HTTP server"},
		{[]string{"python3", "/venv/bin/flask", "run"}, "Flask"},
		{[]string{"python3", "-m", "uvicorn", "main:app"}, "Uvicorn"},
		{[]string{"python3.12", "-m", "uvicorn", "app:app", "--host", "127.0.0.1"}, "Uvicorn"},
		{[]string{"python3", "/venv/bin/uvicorn", "main:app"}, "Uvicorn"},
		{[]string{"python3", "-m", "gunicorn", "main:app"}, "Gunicorn"},
		{[]string{"python3", "/venv/bin/gunicorn", "main:app"}, "Gunicorn"},
		{[]string{"php", "-S", "127.0.0.1:8000"}, "PHP server"},
		{[]string{"node", "/project/node_modules/nuxt/bin/nuxt.mjs", "build"}, ""},
		{[]string{"node", "/project/node_modules/next/dist/bin/next", "build"}, ""},
		{[]string{"node", "/project/node_modules/next/dist/bin/next"}, ""},
		{[]string{"node", "/project/node_modules/next/dist/bin/next", "--version"}, ""},
		{[]string{"node", "/project/node_modules/next/dist/bin/next", "dev", "--help"}, ""},
		{[]string{"node", "/project/node_modules/storybook/dist/bin/dispatcher.js", "dev", "--help"}, ""},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js", "--host", "--version"}, ""},
		{[]string{"node", "/project/node_modules/@nuxt/cli/bin/nuxi.mjs", "build"}, ""},
		{[]string{"node", "/project/node_modules/@angular/cli/bin/ng.js", "test"}, ""},
		{[]string{"node", "/project/node_modules/@nestjs/cli/bin/nest.js", "build"}, ""},
		{[]string{"node", "/project/node_modules/@remix-run/dev/dist/cli.js", "build"}, ""},
		{[]string{"node", "/project/node_modules/storybook/dist/bin/dispatcher.js", "build"}, ""},
		{[]string{"node", "/project/node_modules/expo/bin/cli", "export"}, ""},
		{[]string{"python3", "manage.py", "test"}, ""},
		{[]string{"python3", "-m", "flask", "shell"}, ""},
		{[]string{"python3.12", "-m", "flask", "--app", "hello", "--debug", "run"}, ""},
		{[]string{"python3", "/venv/bin/flask", "--app=hello", "run"}, ""},
		{[]string{"python3", "-m", "flask", "--app", "hello", "shell"}, ""},
		{[]string{"python3", "-m", "flask", "--app=", "run"}, ""},
		{[]string{"python3", "-m", "flask", "--app", "run"}, ""},
		{[]string{"python3", "-m", "http.server", "--help"}, ""},
		{[]string{"node", "/project/node_modules/vite/bin/vite.js", "--help"}, ""},
		{[]string{"python3", "-m", "uvicorn", "--help"}, ""},
		{[]string{"python3", "-m", "uvicorn"}, ""},
		{[]string{"python3.12", "-m", "uvicorn", "--host", "127.0.0.1", "--port=8000", "app:app"}, ""},
		{[]string{"python3", "/venv/bin/uvicorn", "--reload", "--host", "127.0.0.1", "app:app"}, ""},
		{[]string{"python3", "-m", "uvicorn", "--host", "127.0.0.1"}, ""},
		{[]string{"python3", "-m", "uvicorn", "--host", "--port", "8000", "app:app"}, ""},
		{[]string{"python3", "-m", "uvicorn", "--host=", "app:app"}, ""},
		{[]string{"python3", "-m", "uvicorn", "--unknown", "x", "app:app"}, ""},
		{[]string{"python3", "-m", "uvicorn", "app:app", "--help"}, ""},
		{[]string{"python3", "/venv/bin/gunicorn", "--version"}, ""},
		{[]string{"php", "-S"}, ""},
		{[]string{"go", "run", "."}, ""},
		{[]string{"cargo", "run"}, ""},
	}
	for _, tt := range tests {
		if got := catalog(tt.args, "/project"); got != tt.want {
			t.Errorf("%v: got %q want %q", tt.args, got, tt.want)
		}
	}
}

func TestCatalogOtherRuntimes(t *testing.T) {
	entries := []signature{
		{name: "Python", executables: []string{"python3"}, arguments: []string{"-m", "http.server"}},
		{name: "PHP", executables: []string{"php"}, arguments: []string{"-S"}},
	}
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"python3", "-m", "http.server", "8000"}, "Python"},
		{[]string{"php", "-S", "localhost:8000"}, "PHP"},
		{[]string{"python3", "-m", "other"}, ""},
		{[]string{"php", "-v"}, ""},
	} {
		if got := matchCatalog(tt.args, "/project", entries); got != tt.want {
			t.Errorf("%v: got %q want %q", tt.args, got, tt.want)
		}
	}
}

func TestRecognitionRejectsReusedAndCyclicAncestors(t *testing.T) {
	child := process{pid: 11, parent: 10, start: 10}
	parent := process{pid: 10, start: 11, cwd: "/project", args: []string{"node", "/project/node_modules/next/dist/bin/next", "dev"}}
	if _, _, ok := recognized(child, map[int]process{10: parent}); ok {
		t.Fatal("accepted newer parent")
	}
	child.parent = 11
	if _, _, ok := recognized(child, map[int]process{11: child}); ok {
		t.Fatal("accepted cycle")
	}
}

func TestRecognitionRequiresDirectParentAndSameDirectory(t *testing.T) {
	parent := process{pid: 10, start: 1, cwd: "/project", args: []string{"node", "/project/node_modules/next/dist/bin/next", "dev"}}
	child := process{pid: 11, parent: 10, start: 2, cwd: "/other"}
	if _, _, ok := recognized(child, map[int]process{10: parent}); ok {
		t.Fatal("accepted a child in a different directory")
	}
	child.cwd = "/project"
	grandchild := process{pid: 12, parent: 11, start: 3, cwd: "/project"}
	if _, _, ok := recognized(grandchild, map[int]process{10: parent, 11: child}); ok {
		t.Fatal("accepted a distant ancestor")
	}
}

func TestSymlinkedEntrypoint(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "node_modules", ".bin")
	target := filepath.Join(dir, "node_modules", "next", "dist", "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "next"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(target, "next"), filepath.Join(bin, "next")); err != nil {
		t.Fatal(err)
	}
	if got := catalog([]string{"node", filepath.Join(bin, "next"), "dev"}, dir); got != "Next.js" {
		t.Fatal(got)
	}
}

func TestPackageManagerEntrypoints(t *testing.T) {
	for _, layout := range []string{"node_modules", "node_modules/.pnpm/nuxt@4/node_modules"} {
		t.Run(layout, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, layout, "nuxt/bin/nuxt.mjs")
			bin := filepath.Join(dir, "node_modules/.bin/nuxt")
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(bin), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, nil, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, bin); err != nil {
				t.Fatal(err)
			}
			if got := catalog([]string{"node", bin, "dev"}, dir); got != "Nuxt" {
				t.Fatal(got)
			}
		})
	}
}

func TestAncestorAndListeners(t *testing.T) {
	parent := process{pid: 10, cwd: "/project", name: "node", args: []string{"node", "/project/node_modules/next/dist/bin/next", "dev"}}
	child := process{pid: 11, parent: 10, cwd: "/project", name: "next-server"}
	name, evidence, ok := recognized(child, map[int]process{10: parent, 11: child})
	if !ok || name != "Next.js" || evidence.pid != 10 {
		t.Fatalf("not recognized: %q %#v", name, evidence)
	}
	service := Service{Name: name, PID: child.pid, Process: child.name, Project: evidence.cwd}
	listeners := []listener{{"127.0.0.1", 3000, "a"}, {"::1", 3000, "b"}, {"127.0.0.1", 3000, "c"}, {"127.0.0.1", 80, "unknown"}}
	got := collect(listeners, map[string]Service{"a": service, "b": service, "c": service}, nil)
	if len(got) != 1 || len(got[0].Endpoints) != 2 {
		t.Fatalf("unexpected services: %#v", got)
	}
}

func TestDescendantSocketOwnersRemainSeparate(t *testing.T) {
	proc := t.TempDir()
	for _, p := range []struct {
		pid, parent      int
		name, cmd, inode string
	}{
		{10, 1, "node", "node\x00/project/node_modules/next/dist/bin/next\x00dev\x00", ""},
		{11, 10, "worker-a", "worker-a\x00", "111"},
		{12, 10, "worker-b", "worker-b\x00", "222"},
		{13, 1, "other", "other\x00", "333"},
	} {
		base := filepath.Join(proc, fmt.Sprint(p.pid))
		if err := os.MkdirAll(filepath.Join(base, "fd"), 0755); err != nil {
			t.Fatal(err)
		}
		stat := fmt.Sprintf("%d (%s) S %d 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 %d 0", p.pid, p.name, p.parent, p.pid)
		for name, value := range map[string]string{"stat": stat, "comm": p.name, "cmdline": p.cmd} {
			if err := os.WriteFile(filepath.Join(base, name), []byte(value), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink("/project", filepath.Join(base, "cwd")); err != nil {
			t.Fatal(err)
		}
		if p.inode != "" {
			if err := os.Symlink("socket:["+p.inode+"]", filepath.Join(base, "fd", "3")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("socket:[shared]", filepath.Join(base, "fd", "4")); err != nil {
				t.Fatal(err)
			}
		}
	}
	owners := processOwners(proc, os.Getuid())
	if owners["111"].PID != 11 || owners["111"].Process != "worker-a" || owners["222"].PID != 12 {
		t.Fatalf("wrong owners: %#v", owners)
	}
	if _, ok := owners["shared"]; ok {
		t.Fatalf("shared socket attributed to an arbitrary owner: %#v", owners["shared"])
	}
	if _, ok := owners["333"]; ok {
		t.Fatal("unrecognized process accepted")
	}
	got := collect([]listener{{"127.0.0.1", 3000, "111"}, {"::1", 3000, "111"}, {"127.0.0.1", 4000, "222"}}, owners, nil)
	if len(got) != 2 || got[0].PID != 11 || len(got[0].Endpoints) != 2 || got[1].PID != 12 {
		t.Fatalf("incorrect grouping: %#v", got)
	}
}

func TestDockerGroups(t *testing.T) {
	a := container{ID: "a", Names: []string{"/web"}, Labels: map[string]string{"com.docker.compose.project": "demo"}}
	b := container{ID: "b", Names: []string{"/db"}, Labels: map[string]string{"com.docker.compose.project": "demo"}}
	c := container{ID: "c", Names: []string{"/independent"}}
	a.Ports = append(a.Ports, struct {
		IP         string `json:"IP"`
		PublicPort uint16 `json:"PublicPort"`
		Type       string `json:"Type"`
	}{"127.0.0.1", 3000, "tcp"})
	got := collect(nil, nil, []container{a, b, c})
	if len(got) != 2 || got[0].Name != "demo" || len(got[0].Members) != 2 || got[0].Members[0].ID != "a" || got[0].Members[0].Name != "web" || len(got[0].Members[0].Endpoints) != 1 || got[0].Members[1].ID != "b" || got[0].Members[1].Name != "db" || got[1].ContainerID != "c" || got[1].Name != "independent" {
		t.Fatalf("unexpected containers: %#v", got)
	}
	if got := collect(nil, nil, nil); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestDockerCgroupDeduplication(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, line := range []string{"0::/system.slice/docker-" + id + ".scope", "1:name=systemd:/docker/" + id} {
		if got := dockerCgroupID(line); got != id {
			t.Fatalf("%q: %q", line, got)
		}
	}
	if got := dockerCgroupID("0::/system.slice/docker-not-an-id.scope"); got != "" {
		t.Fatal(got)
	}
	listeners := []listener{{"127.0.0.1", 3000, "a"}, {"127.0.0.1", 3000, "b"}}
	owners := map[string]Service{"a": {PID: 1, Name: "Next.js", ContainerID: id}, "b": {PID: 2, Name: "Next.js"}}
	got := collect(listeners, owners, []container{{ID: id, Names: []string{"/web"}}})
	if len(got) != 2 || got[0].PID != 2 || got[1].Name != "web" {
		t.Fatalf("unexpected services: %#v", got)
	}
}

func TestDockerUnixSockets(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/containers/json":
			fmt.Fprint(w, `[{"Id":"abc","Names":["/web"],"Labels":{"com.docker.compose.project":"demo"},"Ports":[{"IP":"127.0.0.1","PublicPort":3000,"Type":"tcp"}]}]`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	t.Setenv("DOCKER_HOST", "unix://"+socket)
	if got := docker(context.Background()); len(got) != 1 || got[0].ID != "abc" || len(got[0].Ports) != 1 {
		t.Fatalf("DOCKER_HOST: %#v", got)
	}
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("XDG_RUNTIME_DIR", dir)
	if got := docker(context.Background()); len(got) != 1 || got[0].ID != "abc" {
		t.Fatalf("rootless: %#v", got)
	}
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:2375")
	if got := docker(context.Background()); len(got) != 0 {
		t.Fatalf("remote host accepted: %#v", got)
	}
}

func TestProcessOwnerFixture(t *testing.T) {
	proc := t.TempDir()
	base := filepath.Join(proc, "42")
	if err := os.MkdirAll(filepath.Join(base, "fd"), 0755); err != nil {
		t.Fatal(err)
	}
	stat := "42 (node) S 1 42 42 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1234 0"
	for name, value := range map[string]string{"stat": stat, "comm": "node", "cmdline": "node\x00/project/node_modules/next/dist/bin/next\x00dev\x00"} {
		if err := os.WriteFile(filepath.Join(base, name), []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("/project", filepath.Join(base, "cwd")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("socket:[123]", filepath.Join(base, "fd", "3")); err != nil {
		t.Fatal(err)
	}
	listeners := []listener{{"127.0.0.1", 3000, "123"}}
	if got := collect(listeners, processOwners(proc, os.Getuid()), nil); len(got) != 1 || got[0].PID != 42 {
		t.Fatalf("unexpected services: %#v", got)
	}
	if got := collect(listeners, processOwners(proc, os.Getuid()+1), nil); len(got) != 0 {
		t.Fatalf("wrong owner accepted: %#v", got)
	}
}

func TestDockerUnavailable(t *testing.T) {
	if got := dockerAt(context.Background(), filepath.Join(t.TempDir(), "missing.sock")); len(got) != 0 {
		t.Fatal(got)
	}
}
