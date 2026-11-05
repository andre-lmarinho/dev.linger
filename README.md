# DevLinger

**What's still running?** DevLinger shows listening TCP ports and running Docker containers in the Linux system tray. It is read-only and does not need an editor, browser, or root access.

## Requirements

- Linux, a session D-Bus, and a desktop panel that supports StatusNotifierItem (some GNOME desktops require an extension).
- Go 1.24 or later to build.
- Optional: permission to access the local Docker socket. Docker access itself grants powerful privileges; DevLinger only reads container information.

```sh
go build -o devlinger .
./devlinger
```

The menu refreshes every 10 seconds. Process names and working directories are shown only when `/proc` permits reading them; a working directory is not necessarily a project root. Docker Compose project labels are used when available. Docker is optional. Published Docker ports are merged with corresponding Docker proxy listeners when identifiable.

The app does not infer HTTP from a port number or probe services, so it does not offer links unless a future version can identify them safely. TCP listeners in other network namespaces may not be visible. Remote Docker contexts and rootless Docker sockets outside the default location are not currently detected. A desktop without a StatusNotifierItem host cannot display the indicator.

## Tests

```sh
go test ./...
```

MIT licensed; see [LICENSE](LICENSE).
