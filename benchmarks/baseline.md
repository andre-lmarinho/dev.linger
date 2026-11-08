# Baseline — 2026-11-06

Application revision `827f1ea` plus the listener inode and deferred recognition changes accompanying this baseline. Linux 7.0.0-42, amd64, Go 1.27.3, AMD Ryzen 5 5600G, 12 logical CPUs. Docker's default socket was present; daemon and container availability were not verified during these measurements. Results describe this environment, not necessarily normal use.

## Stage benchmarks

Method: [README.md](README.md), three samples of at least 500 ms per benchmark. Monitor and tray runs were sequential. The live monitor initially observed 144 user processes, 9 listeners and no recognized services. Its workload could change during the run. Times are elapsed time per operation, not CPU time; `B/op` is allocated Go memory, not RSS. Bytes and allocations below are independent medians.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 23.95 ms | 23.85–24.42 ms | 2,268,802 | 27,060 |
| Live Listeners | 2.361 ms | 2.353–2.362 ms | 38,320 | 154 |
| Live Processes | 5.606 ms | 5.588–5.718 ms | 1,069,859 | 5,600 |
| Live AssociateSockets | 13.82 ms | 13.73–14.00 ms | 1,130,380 | 21,178 |
| Live Docker | 177.2 µs | 176.0–177.6 µs | 18,948 | 118 |
| Live Metrics, no services | 51.18 µs | 51.15–51.25 µs | 15,848 | 23 |
| Live MetricsSelf, one process | 76.57 µs | 76.01–76.72 µs | 21,248 | 45 |
| Catalog, Vite symlink | 19.58 µs | 19.52–19.64 µs | 3,808 | 45 |
| Catalog, unrecognized command | 18.83 µs | 18.82–18.93 µs | 3,664 | 44 |
| Tray.Update, private watcher | 38.14 µs | 37.94–38.29 µs | 15,599 | 503 |

Stages are independently measured, not a breakdown of the same cycle. The catalog inputs are synthetic; the tray does not include a real desktop watcher or menu interaction.

## Synthetic fixture

`go test ./internal/monitor -run '^$' -bench 'Benchmark(Live|Catalog|SyntheticProc)' -benchmem -benchtime=500ms -count=3 -v` also measured the fixed fixture described in [README.md](README.md). It uses ordinary files, not kernel `/proc`.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Listeners | 19.83 µs | 19.74–19.84 µs | 12,704 | 49 |
| Processes | 2.124 ms | 2.106–2.124 ms | 332,752 | 2,627 |
| AssociateSockets | 2.108 ms | 2.105–2.132 ms | 310,877 | 4,843 |
| Discovery | 4.422 ms | 4.404–4.448 ms | 675,944 | 7,720 |

## Recognized service and whole process

A separate run used a local `python3 -m http.server 39871 --bind 127.0.0.1`. The benchmark initially observed 145 user processes, 10 listeners and one recognized service. Three samples of at least 500 ms per stage yielded:

| Benchmark | Median time | Range | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 24.12 ms | 24.00–24.12 ms | 2,286,508 | 27,156 |
| Live Metrics | 75.47 µs | 74.47–75.53 µs | 21,216 | 45 |

With the server still running, two separate private-bus runs measured the release-style application (`-trimpath -ldflags='-s -w'`) using the [process sampler](README.md#whole-process-cpu-and-rss). Each lasted 35 seconds including startup. The sampler did not count completed scans; the host workload and Docker state were not controlled.

| Run | Sampled interval | CPU time | Avg CPU, one core | Final RSS | Peak RSS (`VmHWM`) |
|---|---:|---:|---:|---:|---:|
| 1 | 34.852 s | 0.10 s | 0.287% | 15,860 KiB | 15,860 KiB |
| 2 | 34.849 s | 0.10 s | 0.287% | 15,212 KiB | 15,472 KiB |

CPU time is quantized to 10 ms clock ticks; the first sample can miss startup CPU. RSS includes shared pages. Two short runs cannot establish long-term memory usage, and the private watcher excludes desktop-side resources.
