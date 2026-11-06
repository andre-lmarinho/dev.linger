# DevLinger

**What's still running?**

DevLinger is a read-only Linux tray app for finding your development servers and running Docker containers, even after you close your editor.

## What it shows

- Recognized local development servers, with their project, ports, Git branch, uptime, CPU and memory when available.
- Running Docker containers, grouped by Compose project when applicable.
- A single tray menu that refreshes every 10 seconds. DevLinger does not stop or modify services.

Unknown listeners are omitted. Docker resource usage and uptime are not collected.

## Get started

You need Linux, a D-Bus session and a StatusNotifierItem-compatible tray. On GNOME, an AppIndicator extension may be required. Docker is optional; access to its socket grants powerful privileges.

### Download

When available, download the Linux amd64 binary or Debian package from [GitHub Releases](https://github.com/andre-lmarinho/dev.linger/releases). The Debian package installs the `devlinger` executable; it does not add an application launcher.

### Build from source

Requires Go 1.24 or later:

```sh
go build -o devlinger .
./devlinger
```

Open the tray icon to see services. Select **Quit** to exit.

## Detection

DevLinger recognizes explicit commands for Next.js, Astro, Vite, Nuxt, Angular, NestJS, Remix, Storybook, Expo, Python HTTP Server, Django, Flask, Uvicorn and PHP's built-in server. See [`internal/monitor/catalog.go`](internal/monitor/catalog.go) for the supported entrypoints and argument order.

Detection requires an active TCP listener and can miss services with unsupported command layouts or shared sockets. For example, Gunicorn's usual master/worker socket is currently omitted. CPU usage is calculated between scans; 100% means one fully used core.

Docker discovery supports local Unix sockets only. See the [Docker socket selection](#docker-socket) note below if containers are missing.

### Docker socket

DevLinger uses an absolute Unix `DOCKER_HOST` when set. Otherwise it checks `$XDG_RUNTIME_DIR/docker.sock`, then `/var/run/docker.sock`. If the selected socket is inaccessible, it does not fall back to another. Remote hosts and Docker CLI contexts are not supported.

## Development

```sh
dbus-run-session -- go test -race ./...
go vet ./...
```

Licensed under the [MIT License](LICENSE).
