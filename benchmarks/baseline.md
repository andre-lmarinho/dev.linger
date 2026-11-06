# Baseline — 2026-11-06

Application revision `97e5e88`, unchanged in benchmark commit `22f2d52`; no optimizations. Some benchmark fixtures and the process sampler were added after that commit, as noted below. Linux 7.0.0-42, amd64, Go 1.27.3, AMD Ryzen 5 5600G, 12 logical CPUs. The default Docker socket was present; daemon and container availability were not checked. These runs do not necessarily represent normal usage.

Method: commands in [README.md](README.md), three samples of at least 500 ms, run sequentially by package. The monitor initially observed 146 user processes, 8 listeners and no recognized services. A separate snapshot found 396 system processes, 149 user processes and 3,453 accessible user file descriptors. The workload varied; these counts were not simultaneous with the baseline.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Scan | 42.90 ms | 42.72–43.09 ms | 5,483,130 | 68,031 |
| Listeners | 2.30 ms | 2.30–2.34 ms | 27,712 | 106 |
| Processes | 5.70 ms | 5.57–5.89 ms | 1,072,063 | 5,592 |
| AssociateSockets | 32.75 ms | 32.68–33.23 ms | 4,358,075 | 62,183 |
| Docker | 178 µs | 178–182 µs | 18,957 | 118 |
| Metrics, no services | 50.8 µs | 50.7–50.9 µs | 15,848 | 23 |
| MetricsSelf, one process | 76.4 µs | 75.9–76.8 µs | 21,248 | 45 |
| Catalog, Vite symlink | 77.0 µs | 76.7–77.9 µs | 15,232 | 180 |
| Catalog, unrecognized command | 173 µs | 173–174 µs | 36,112 | 419 |
| Tray.Update, fake watcher | 38.4 µs | 37.7–39.4 µs | 15,598 | 503 |

Bytes and allocations are independent medians; rounded times can obscure differences. Isolated stages are not an exact breakdown of the same cycle. In a separate `AssociateSockets` profile, `filepath.walkSymlinks` accounted for 43% of cumulative CPU samples; system calls dominated the profile. This warrants investigating repeated symlink resolution but does not estimate the gain from a change. The unrecognized catalog case is synthetic.

The `go build -trimpath -ldflags='-s -w'` binary was 8,470,791 bytes; without stripping it was 12,466,639 bytes. The unstripped symbol listing includes `net/http` and TLS code but does not directly attribute file size to packages. BSS symbols do not occupy equivalent space in the file.

## One recognized service

A separate run used a local `python3 -m http.server` bound to loopback. The benchmark initially observed 148 user processes, 9 listeners and one recognized service. The same Go benchmark settings were used, with three samples per stage:

| Benchmark | Median time | Range | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Scan | 44.58 ms | 44.17–44.75 ms | 5,555,196 | 68,740 |
| Metrics.Update | 75.64 µs | 75.30–76.71 µs | 21,216 | 45 |

To reproduce, start `python3 -m http.server 39871 --bind 127.0.0.1` in a separate terminal, run `go test ./internal/monitor -run '^$' -bench '^BenchmarkLive/(Scan|Metrics)$' -benchmem -benchtime=500ms -count=3 -v`, and stop the server. Check the fixture log confirms a recognized service; existing workloads may add others. `Tray.Update` remains the private-bus measurement above, not a real-desktop measurement. These independent runs are not a controlled before/after comparison.

## Limits and next steps

- `AssociateSockets` is the main candidate in this scenario; measure under real workloads and verify symlink, recognition and shared-socket correctness before proposing a change.
- Measure real projects, more listeners, Docker active/inactive and high load.
- CPU and RSS were measured only in the short private-bus scenario below; real-desktop resource use and D-Bus behavior on Zorin, including `AboutToShow` and an open menu, remain unmeasured. There is no conclusion about user experience or expected gains.
- No optimizations, frequency changes or CI configuration were made.

## Synthetic fixture reference

The fixed fixture described in [README.md](README.md) was run separately with `go test ./internal/monitor -run '^TestSyntheticProc$' -bench '^BenchmarkSyntheticProc$' -benchmem -benchtime=500ms -count=3`. Same machine and Go as above; the fixture test passed. These measurements are a reference for the fixture, not a before/after comparison or a substitute for real `/proc`.

| Benchmark | Median time | Range | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Listeners | 20.50 µs | 20.42–20.85 µs | 12,720 | 49 |
| Processes | 2.222 ms | 2.220–2.227 ms | 332,821 | 2,627 |
| AssociateSockets | 13.30 ms | 13.22–13.64 ms | 2,628,588 | 33,260 |
| Discovery | 15.56 ms | 15.56–15.57 ms | 2,992,182 | 36,137 |

## Whole-process CPU and RSS

Two baseline runs of the unchanged application used `-trimpath -ldflags='-s -w'`, a private D-Bus watcher and a local Python HTTP server. A separate check observed 141 user processes, 9 listeners and one recognized service; host workload and Docker state were not controlled. The [sampler](README.md#whole-process-cpu-and-rss) measured approximately 35 seconds per run, including startup, with samples roughly every 200 ms. Runs were interleaved with a separate experimental candidate; only committed-application results appear here. The sampler and watcher were added after `22f2d52`.

| Run | Sampled interval | CPU time | Avg CPU, one core | Final RSS | Peak RSS (`VmHWM`) |
|---|---:|---:|---:|---:|---:|
| 1 | 34.849 s | 0.18 s | 0.517% | 15,796 KiB | 15,796 KiB |
| 2 | 34.850 s | 0.19 s | 0.545% | 15,808 KiB | 15,808 KiB |

CPU time is quantized to 10 ms clock ticks; the first sample may miss startup CPU. RSS includes shared pages. Two short runs cannot establish long-term memory usage or represent the real desktop. The sampler does not count completed scans, and the private watcher excludes desktop-side resources.
