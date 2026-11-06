# Baseline — 2026-11-06

Application revision `e150754` plus the catalog path-resolution change accompanying this baseline. Linux 7.0.0-42, amd64, Go 1.27.3, AMD Ryzen 5 5600G, 12 logical CPUs. Docker's default socket was present; daemon and container availability were not verified. Results describe this environment, not necessarily normal use.

## Stage benchmarks

Method: [README.md](README.md), three samples of at least 500 ms per benchmark. Monitor and tray runs were sequential. The live monitor initially observed 142 user processes, 8 listeners and no recognized services. Its workload could change during the run. Times are elapsed time per operation, not CPU time; `B/op` is allocated Go memory, not RSS. Bytes and allocations below are independent medians.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 28.03 ms | 27.81–28.05 ms | 2,882,035 | 31,045 |
| Live Listeners | 2.362 ms | 2.351–2.369 ms | 35,841 | 143 |
| Live Processes | 6.032 ms | 5.537–6.130 ms | 1,059,952 | 5,516 |
| Live AssociateSockets | 18.09 ms | 18.00–18.11 ms | 1,768,587 | 25,404 |
| Live Docker | 176.7 µs | 174.8–176.8 µs | 18,960 | 118 |
| Live Metrics, no services | 51.22 µs | 50.61–51.38 µs | 15,848 | 23 |
| Live MetricsSelf, one process | 76.57 µs | 76.36–77.29 µs | 21,248 | 45 |
| Catalog, Vite symlink | 19.48 µs | 19.28–19.53 µs | 3,872 | 45 |
| Catalog, unrecognized command | 18.74 µs | 18.70–18.77 µs | 3,712 | 44 |
| Tray.Update, private watcher | 37.51 µs | 37.20–38.32 µs | 15,598 | 503 |

Stages are independently measured, not a breakdown of the same cycle. The catalog inputs are synthetic; the tray does not include a real desktop watcher or menu interaction.

## Synthetic fixture

`go test ./internal/monitor -run '^$' -bench 'Benchmark(Live|Catalog|SyntheticProc)' -benchmem -benchtime=500ms -count=3 -v` also measured the fixed fixture described in [README.md](README.md). It uses ordinary files, not kernel `/proc`.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Listeners | 19.78 µs | 19.77–19.79 µs | 12,704 | 49 |
| Processes | 2.140 ms | 2.123–2.159 ms | 332,701 | 2,627 |
| AssociateSockets | 3.094 ms | 3.086–3.103 ms | 474,325 | 6,737 |
| Discovery | 5.429 ms | 5.409–5.431 ms | 839,574 | 9,615 |

## Recognized service and whole process

A separate run used a local `python3 -m http.server 39871 --bind 127.0.0.1`. The benchmark initially observed 144 user processes, 9 listeners and one recognized service. Three samples of at least 500 ms per stage yielded:

| Benchmark | Median time | Range | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 28.14 ms | 28.14–28.16 ms | 2,913,106 | 31,321 |
| Live Metrics | 75.22 µs | 74.98–75.66 µs | 21,216 | 45 |

With the server still running, two separate private-bus runs measured the release-style application (`-trimpath -ldflags='-s -w'`) using the [process sampler](README.md#whole-process-cpu-and-rss). Each lasted 35 seconds including startup. The sampler did not count completed scans; the host workload and Docker state were not controlled.

| Run | Sampled interval | CPU time | Avg CPU, one core | Final RSS | Peak RSS (`VmHWM`) |
|---|---:|---:|---:|---:|---:|
| 1 | 34.849 s | 0.12 s | 0.344% | 15,396 KiB | 15,396 KiB |
| 2 | 34.852 s | 0.13 s | 0.373% | 15,440 KiB | 15,668 KiB |

CPU time is quantized to 10 ms clock ticks; the first sample can miss startup CPU. RSS includes shared pages. Two short runs cannot establish long-term memory usage, and the private watcher excludes desktop-side resources.

