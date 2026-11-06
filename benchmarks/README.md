# Performance benchmarks

Run from the repository root on Linux. Go 1.24+ is required, as is `dbus-run-session` for the tray. There is no CI integration or automatic threshold.

```sh
go test ./...
go test ./internal/monitor -run '^$' -bench 'Benchmark(Live|Catalog)' -benchmem -benchtime=500ms -count=3 -v
dbus-run-session -- go test ./internal/tray -run '^$' -bench '^BenchmarkUpdate$' -benchmem -benchtime=500ms -count=3
```

`Live` reads the real `/proc` and accesses the Docker socket selected by the application. Its log reports user processes, listeners and services at startup; the workload can change during a run. `Metrics` uses discovered services; `MetricsSelf` measures one valid process. `AssociateSockets` uses an initial process snapshot but reads real file descriptors on every iteration. The catalog uses temporary files. The tray uses a fake watcher on a private bus: it does not measure the real desktop or opening the menu. Do not run the tray benchmark on the desktop bus.

`ns/op` is average elapsed time per iteration, not CPU time; `B/op` and `allocs/op` are Go allocations, not RSS. To compare versions, use the same Go version, machine, permissions, Docker state and workload; run sequentially, repeat and record variation. `/proc` results are nondeterministic and unsuitable for a strict CI threshold without a controlled environment. Avoid publishing output or profiles that expose paths, processes or other local data.

For further investigation:

```sh
go test ./internal/monitor -run '^$' -bench '^BenchmarkLive/AssociateSockets$' -benchtime=3s -cpuprofile /tmp/monitor-cpu.pprof
go tool pprof -top /tmp/monitor-cpu.pprof
go build -trimpath -ldflags='-s -w' -o /tmp/devlinger-release .
stat -c '%s bytes' /tmp/devlinger-release
```

Profiling and tracing add overhead; do not compare their timings with normal runs. To inspect symbols, build separately without `-ldflags='-s -w'` and use `go tool nm -size -sort size`; BSS symbols do not correspond to bytes in the file. This method does not replace measurements of the complete application (CPU, RSS and D-Bus on the real desktop).

Recorded results: [2026-11-06 baseline](baseline-2026-11-06.md).
