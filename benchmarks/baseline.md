# Baseline — 2026-11-17

Application revision `fa6172b`. Linux 7.0.0-43, amd64, Go 1.27.3, AMD Ryzen 5 5600G, 12 logical CPUs. Docker's default socket was present; daemon and container availability were not otherwise verified. Results describe this environment, not necessarily normal use.

## Stage benchmarks

Method: [README.md](README.md), three samples of at least 500 ms per benchmark. Monitor and tray runs were sequential. The live monitor initially observed 217 user processes, eight listeners and no recognized services. Its workload could change during the run. Times are elapsed time per operation, not CPU time; `B/op` is allocated Go memory, not RSS. Bytes and allocations below are independent medians.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 32.26 ms | 31.99–32.31 ms | 3,010,821 | 35,314 |
| Live Listeners | 2.336 ms | 2.333–2.369 ms | 28,480 | 110 |
| Live Processes | 9.180 ms | 9.176–10.00 ms | 1,478,781 | 7,556 |
| Live AssociateSockets | 20.16 ms | 18.42–20.44 ms | 1,494,221 | 27,758 |
| Live Docker | 249.2 µs | 240.6–250.5 µs | 18,955 | 118 |
| Catalog, Vite symlink | 19.24 µs | 18.99–19.42 µs | 3,872 | 45 |
| Catalog, unrecognized command | 18.58 µs | 18.46–18.77 µs | 3,712 | 44 |
| Tray.Update, private watcher | 39.44 µs | 38.65–40.17 µs | 16,754 | 506 |

Stages are independently measured, not a breakdown of the same cycle. The catalog inputs are synthetic; the tray does not include a real desktop watcher or menu interaction. CPU, memory and uptime collection no longer exist as separate stages.

## Synthetic fixture

`go test ./internal/monitor -run '^$' -bench 'Benchmark(Live|Catalog|SyntheticProc)' -benchmem -benchtime=500ms -count=3 -v` also measured the fixed fixture described in [README.md](README.md). It uses ordinary files, not kernel `/proc`.

| Benchmark | Median time | Range of 3 samples | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Listeners | 20.62 µs | 20.50–21.27 µs | 12,720 | 49 |
| Processes | 2.184 ms | 2.179–2.191 ms | 332,790 | 2,627 |
| AssociateSockets | 2.255 ms | 2.216–2.265 ms | 313,503 | 4,833 |
| Discovery | 4.653 ms | 4.581–4.753 ms | 664,021 | 7,530 |

## Recognized service and whole process

A separate run used a local `python3 -m http.server 39871 --bind 127.0.0.1`. The benchmark initially observed 222 user processes, nine listeners and one recognized service. Three samples of at least 500 ms yielded:

| Benchmark | Median time | Range | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Live Scan | 31.23 ms | 30.92–31.35 ms | 3,087,573 | 36,135 |

With the server still running, two separate private-bus runs measured the release-style application (`-trimpath -ldflags='-s -w'`) using the [process sampler](README.md#whole-process-cpu-and-rss). The binary was 8,495,367 bytes. Each run lasted 35 seconds including startup. The sampler did not count completed scans; the host workload and Docker state were not controlled.

| Run | Sampled interval | CPU time | Avg CPU, one core | Final RSS | Peak RSS (`VmHWM`) |
|---|---:|---:|---:|---:|---:|
| 1 | 34.85 s | 0.14 s | 0.402% | 16,128 KiB | 16,128 KiB |
| 2 | 34.85 s | 0.13 s | 0.373% | 15,756 KiB | 15,756 KiB |

CPU time is quantized to 10 ms clock ticks; the first sample can miss startup CPU. RSS includes shared pages. Two short runs cannot establish long-term memory usage, and the private watcher excludes desktop-side resources.
