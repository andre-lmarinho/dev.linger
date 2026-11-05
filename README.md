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

The menu shows project, stack, ports, Git branch, uptime, CPU and memory when available. Local services show process CPU and resident memory; CPU is calculated between scans and can exceed 100% when multiple cores are used. Docker uptime and resource metrics are not collected. Container and sub-service entries do not show uptime or metrics.

## Detection

DevLinger recognizes explicit development commands for Next.js, Astro, Vite, Nuxt, Angular, NestJS, Remix, Storybook and Expo; Python HTTP Server, Django, Flask and Uvicorn; and PHP's built-in server. Docker Compose containers are grouped by project. Listeners are grouped by their owning process; a direct parent's command can provide evidence when both share a working directory. Unknown processes are omitted, not assumed irrelevant.

Detection is limited to the entry points and argument order in `internal/monitor/catalog.go`. Package layouts, custom interpreter names, unsupported options before the server command and detached processes can prevent detection. Vite-based SvelteKit is identified as Vite, not SvelteKit. Laravel, Rails, Phoenix, Go, Rust, Spring Boot and ASP.NET Core are not recognized by their generic runtimes alone. Gunicorn signatures exist, but its usual master/worker processes share a socket, which is currently omitted. Other shared sockets are also omitted rather than assigned to an arbitrary owner. Docker containers are listed even without published ports; Docker is optional.

Flask options before `run` and Uvicorn or Gunicorn options before the application argument prevent detection. For example, use `FLASK_APP=hello python -m flask run` or `python -m uvicorn app:app --host 127.0.0.1`.

Docker discovery uses an absolute Unix `DOCKER_HOST` if set, otherwise `$XDG_RUNTIME_DIR/docker.sock` if present, or `/var/run/docker.sock`. An inaccessible selected socket does not trigger a fallback. Remote hosts and Docker CLI contexts are not supported. Docker/local duplicates are suppressed only when cgroup metadata identifies the container.

To extend the catalog, add a verified entry point and its accepted development commands in `internal/monitor/catalog.go`, with positive and negative tests.

## Development

```sh
go test ./...
```

MIT licensed; see [LICENSE](LICENSE).
