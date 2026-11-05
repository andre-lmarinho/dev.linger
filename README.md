# DevLinger

**What's still running?**

A lightweight Linux tray app for finding local development servers and running Docker containers. DevLinger is read-only and works independently of your editor.

## Requirements

- Linux with a D-Bus session and a StatusNotifierItem-compatible tray (GNOME may need an AppIndicator extension).
- Go 1.24 or later to build.
- Optional: access to the Docker Unix socket. Docker socket access grants powerful privileges.

## Run

```sh
go build -o devlinger .
./devlinger
```

The tray menu refreshes every 10 seconds. Select **Quit** to exit.

## Detection

DevLinger recognizes Next.js, Astro and Vite development commands and groups Docker Compose containers by project. It groups listeners by their owning process and can use a direct parent's command as evidence when both have the same working directory. Unknown processes are omitted, not assumed irrelevant.

Detection is limited to the entry points in `internal/monitor/catalog.go`. Other stacks, runtimes, package layouts and detached processes may be missed. A direct child of a recognized command can be shown even if it is unrelated. Sockets shared by multiple accessible processes are omitted rather than assigned to an arbitrary owner. Docker containers are listed even without published ports; Docker is optional.

Docker discovery uses an absolute Unix `DOCKER_HOST` if set, otherwise `$XDG_RUNTIME_DIR/docker.sock` if present, or `/var/run/docker.sock`. An inaccessible selected socket does not trigger a fallback. Remote hosts and Docker CLI contexts are not supported. Docker/local duplicates are suppressed only when cgroup metadata identifies the container.

To extend the catalog, add a verified entry point and its accepted development commands in `internal/monitor/catalog.go`, with positive and negative tests.

## Development

```sh
go test ./...
```

MIT licensed; see [LICENSE](LICENSE).
