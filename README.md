# DevLinger

**What's still running?**

DevLinger is a Linux tray app that finds and stops supported development servers and running Docker containers after you close your editor.

## Install

You need Linux, a session D-Bus, and a tray that provides a StatusNotifierWatcher. GNOME may need the AppIndicator extension. Docker is optional. Access to the Docker socket gives control over the Docker daemon and its containers.

### Debian or Ubuntu (amd64)

The signed APT repository is live and currently serves version `0.4.1`. Install with:

```sh
curl -fsSL https://andre-lmarinho.github.io/dev.linger/install.sh | bash
```

The installer configures the APT repository, installs DevLinger, and starts it when it detects a desktop session. It is generated from [`scripts/install.sh.in`](scripts/install.sh.in) with the identity values in [`metadata.sh`](metadata.sh). Review the script before you run it if you prefer. If DevLinger is already running, the installer does not start another instance.

To update DevLinger, run `sudo apt update && sudo apt upgrade`, then restart the app. To remove it, run `sudo apt remove devlinger`. Remove `/etc/apt/sources.list.d/devlinger.list` and `/usr/share/keyrings/devlinger.gpg` separately if you no longer want the repository.

If you installed the `v0.1.0` DEB, run the same installation command once to add the repository and upgrade the package.

### Other Linux packages

The [current `v0.4.1` release](https://github.com/andre-lmarinho/dev.linger/releases/tag/v0.4.1) contains:

- `devlinger_0.4.1_amd64.deb` with SHA-256 `39f2d020e5d8d0341e8d63f55c1273bb4f6947ef51d5731ff83271ecfe184509`
- `devlinger-0.4.1-1.x86_64.rpm` with SHA-256 `eb04fd5b87a4c787d60c44a4835734949e73fff635ab86aaee05d672cb05b28d`
- `DevLinger-0.4.1-x86_64.AppImage` with SHA-256 `e1f70cd9daed202921d1c7c9aec51ab8a424e362cb7596f0294662ff59a31c39`
- `SHA256SUMS`

Install the DEB or RPM with your package manager. To remove the RPM, run `sudo rpm -e devlinger`. To run the AppImage, make it executable and launch it.

The AUR package is not published yet. Arch users can build it from [`packaging/aur`](packaging/aur).

### Build from source

You need Go 1.24 or later:

```sh
go build -o devlinger .
./devlinger
```

## What it shows

- Recognized local development servers, with their project and ports.
- Running Docker containers, grouped by Compose project when applicable.
- A tray menu that refreshes every 10 seconds.

Unknown listeners do not appear.

## Detection and stopping

DevLinger reads active TCP listeners from `/proc` and matches them to recognized processes owned by your user. It omits unknown commands, sockets shared by multiple processes, and processes it cannot read.

Recognized commands include Next.js, Astro, Vite, Nuxt, Angular, NestJS, Remix, Storybook, Expo, Python HTTP Server, Django, Flask, Uvicorn, Gunicorn, and PHP's built-in server. See [`internal/monitor/catalog.go`](internal/monitor/catalog.go) for supported entrypoints and argument order.

When you stop a process, DevLinger checks that its process ID still refers to the same process, then sends it `SIGTERM`. It does not signal the whole process tree or escalate to `SIGKILL`. When you stop a Docker entry, DevLinger asks Docker to stop the container or each container in its Compose project. It does not force-stop containers after a timeout.

### Docker socket

Docker discovery supports local Unix sockets only. Remote hosts and Docker CLI contexts are not supported. If `DOCKER_HOST` is set, it must use the `unix:///absolute/path` form. Otherwise, DevLinger uses `$XDG_RUNTIME_DIR/docker.sock` when that path is a socket, then `/var/run/docker.sock`. It does not try another socket if the selected one is inaccessible.

Docker is optional. Treat access to its socket as privileged.

## Autostart

DEB, RPM, and AUR package installations enable autostart by default with `/etc/xdg/autostart/io.github.devlinger.DevLinger.desktop`.

To disable autostart for your user, copy the system-wide desktop entry to `~/.config/autostart/io.github.devlinger.DevLinger.desktop`. Add `Hidden=true` under `[Desktop Entry]`.

## Development

```sh
dbus-run-session -- go test -race ./...
go vet ./...

# After installing nFPM v2.43.0, test the DEB, RPM, AUR, and AppImage layouts.
scripts/test-package-layouts.sh
```

## Performance

See the [performance benchmarks and baseline](benchmarks/README.md).

## Release process

Each push to `main` checks whether `VERSION` is a newer semantic version than the published tags. If so, the workflow runs verification, publishes the signed APT repository to `gh-pages`, builds the DEB, RPM, and AppImage, then uploads them with `SHA256SUMS` to the same GitHub Release. A push without a version bump publishes nothing.

If a release fails, inspect and rerun **Publish Linux release** in GitHub Actions. A retry is safe while the tag points to the same commit. Do not casually change published tags, `gh-pages`, or the signing key. Existing installations rely on them.

Licensed under the [MIT License](LICENSE).
