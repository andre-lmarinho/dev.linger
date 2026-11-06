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

The versioned [baseline](baseline.md) records measurements of a specific application revision. Keep implementation experiments, patches and raw results under `benchmarks/experiments/`; this directory is ignored by Git and is local to each checkout. Create it when needed. Do not put the only copy of anything that must be shared or retained there.

## Whole-process CPU and RSS

On Linux, build the app and the private watcher with the same Go version and flags for both revisions. Run the sampler only inside a new private bus:

```sh
go build -trimpath -ldflags='-s -w' -o /tmp/devlinger-before .
go build -trimpath -o /tmp/devlinger-watcher ./benchmarks/watcher
dbus-run-session -- python3 benchmarks/process_sample.py /tmp/devlinger-watcher /tmp/devlinger-before --seconds 35
```

Build the candidate separately, keep the host workload and Docker configuration stable, and alternate before/after order. The sampler reports JSON: CPU time and average percentage of one core between its first and last samples, final and sampled maximum RSS, and Linux `VmHWM` (peak RSS) in KiB. CPU is quantized by Linux clock ticks; RSS includes shared resident pages. The sampler checks for early exit, but does not count completed scans. A 35-second run covers startup and approximately four scheduled scans; verify the workload separately, repeat for longer, and report all samples. The fake watcher excludes real desktop CPU, memory and interactions, and cannot establish behavior on Zorin. Do not compare these results directly to Go allocation counts or claim whole-system resource savings.

## Controlled comparison

`BenchmarkSyntheticProc` creates a fixed `/proc`-like tree: 100 user-owned processes, 10 recognized Python listeners, seven descriptors per process and two additional descriptors for one shared socket. A test verifies recognition and shared-socket rejection. `Discovery` combines listener parsing, owner lookup and collection, but excludes Docker, metrics and tray. Filesystem behavior, scheduling, cache warmth and temporary path lengths still vary; this is not a simulation of kernel `/proc` or the real workload.

First ensure **both revisions contain the same benchmark and fixture**, and that the benchmark test passes on each. Use two separate worktrees; substitute their absolute paths below. Build with the same Go version, environment and build flags. Keep the machine idle and use the same filesystem for temporary files (`TMPDIR`) on both runs. This example alternates execution order and preserves raw output:

```sh
before=/absolute/path/to/before
after=/absolute/path/to/after
mkdir -p /tmp/devlinger-comparison
(cd "$before" && go test ./internal/monitor -run '^TestSyntheticProc$' && go test -c -o /tmp/devlinger-comparison/before.test ./internal/monitor)
(cd "$after" && go test ./internal/monitor -run '^TestSyntheticProc$' && go test -c -o /tmp/devlinger-comparison/after.test ./internal/monitor)
for i in 1 2 3 4 5; do
  if [ $((i % 2)) -eq 1 ]; then order="before after"; else order="after before"; fi
  for version in $order; do
    /tmp/devlinger-comparison/$version.test -test.run '^$' -test.bench '^BenchmarkSyntheticProc$' -test.benchmem -test.benchtime=1s > "/tmp/devlinger-comparison/$version-$i.txt"
  done
done
```

Record revisions, fixture version, machine, Go version, `TMPDIR`, workload and all samples. Compare the distributions of `ns/op`, `B/op` and `allocs/op` per benchmark, not a single best run; use a statistical comparison tool such as `benchstat` if available. A change smaller than run-to-run noise is inconclusive. Recheck promising changes with `BenchmarkLive` and the complete application; don't claim CPU or RSS savings from synthetic elapsed time or Go allocation counts alone. Never change the fixture between before and after runs.
