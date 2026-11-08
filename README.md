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

### Install on Debian or Ubuntu (amd64)

After the APT repository is published, install with:

```sh
curl -fsSL https://andre-lmarinho.github.io/dev.linger/install.sh | bash
```

This downloads and runs the [installation script](scripts/install.sh), which configures the signed APT repository, installs DevLinger and starts it in the current desktop session. Review the script before running it if you prefer. If DevLinger is already running, it will not start another instance. Updates are installed with `sudo apt update && sudo apt upgrade`; restart DevLinger to run the new version. To remove the application, run `sudo apt remove devlinger`; remove `/etc/apt/sources.list.d/devlinger.list` and `/usr/share/keyrings/devlinger.gpg` separately if you no longer want this repository.

Already installed the `v0.1.0` `.deb`? Run the same installation command once to add the repository and upgrade the package.

### Install from releases

Each release attaches the same versioned files: `devlinger_0.3.0_amd64.deb`, `devlinger-0.3.0.x86_64.rpm`, `DevLinger-0.3.0-x86_64.AppImage` and `SHA256SUMS`. Verify the download with `sha256sum -c` and install the format by hand, or run the AppImage directly. On Arch, build the package from [`packaging/aur`](packaging/aur) until it is published to the AUR. To remove the RPM, run `sudo rpm -e devlinger`; delete the AppImage file instead.

### Build from source

Requires Go 1.24 or later:

```sh
go build -o devlinger .
./devlinger
```

Open the tray icon to see services. The tray menu is read-only.

## Detection

DevLinger recognizes explicit commands for Next.js, Astro, Vite, Nuxt, Angular, NestJS, Remix, Storybook, Expo, Python HTTP Server, Django, Flask, Uvicorn and PHP's built-in server. See [`internal/monitor/catalog.go`](internal/monitor/catalog.go) for the supported entrypoints and argument order.

Detection requires an active TCP listener and can miss services with unsupported command layouts or shared sockets. For example, Gunicorn's usual master/worker socket is currently omitted. CPU usage is calculated between scans; 100% means one fully used core.

Docker discovery supports local Unix sockets only. See the [Docker socket selection](#docker-socket) note below if containers are missing.

### Docker socket

DevLinger uses an absolute Unix `DOCKER_HOST` when set. Otherwise it checks `$XDG_RUNTIME_DIR/docker.sock`, then `/var/run/docker.sock`. If the selected socket is inaccessible, it does not fall back to another. Remote hosts and Docker CLI contexts are not supported.

## Autostart

Package installations enable autostart by default (`/etc/xdg/autostart/io.github.devlinger.DevLinger.desktop` installed by the DEB, RPM and AUR packages) so DevLinger starts with your login session.

To disable autostart for your user, copy the system-wide Desktop Entry to `~/.config/autostart/io.github.devlinger.DevLinger.desktop` and add `Hidden=true` under `[Desktop Entry]`.

## Development

```sh
dbus-run-session -- go test -race ./...
go vet ./...
```

Performance benchmarks and baseline: [`benchmarks/README.md`](benchmarks/README.md).

Every push to `main` with a newer `VERSION` file publishes a release after tests pass: DEB, RPM, AppImage and checksums land on the same GitHub Release, and the signed APT repository updates. The version comes from the `VERSION` file (always semver, always newer than the published tag); a push without a version bump publishes nothing. If a release fails, check and rerun **Publish Linux release** in GitHub Actions; retries are safe while the tag still points at the same commit. Do not casually change published tags, `gh-pages`, or the signing key; existing installations rely on them.

Licensed under the [MIT License](LICENSE).
