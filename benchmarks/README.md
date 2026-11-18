# Performance benchmarks

DevLinger's benchmark is a **budget contract**, not a table of timings.

Any person who checks out this repository and runs `go test ./...` enforces the same budgets and gets the same numbers — on a US$ 200 laptop, on a US$ 40,000 workstation, anywhere. The indicators are hardware-independent by construction:

| Indicator | Reproducibility | Enforced by |
|---|---|---|
| `allocs/op` | Exact | `TestDiscoveryBudget`, `TestLayoutBudget` |
| `B/op` | Reproducible within a few % with GC state; the ceiling is the contract | `TestDiscoveryBudget`, `TestLayoutBudget` |
| Release binary bytes | Exact for the same Go toolchain | `benchmarks/compare.py` |
| Fixture outputs (services discovered) | Exact | `TestControlledWorkload` |

Elapsed time (`ns/op`) is machine-local. It is **not** a DevLinger indicator: it is never recorded, never published, never compared. Two machines cannot produce similar timings, so timings have no place in this contract. Go's own benchmark output prints them; ignore them. Manual profiling stays possible (`go test -bench`), but it is diagnosis, not benchmarking.

## Fixed workload

Every measurement runs on the same synthetic input, created per run — the host's `/proc`, Docker daemon and desktop session are never read:

- 200 user processes
- 20 TCP listeners
- 3 recognized development servers owning 10 listeners
- 5 listeners owned by unrecognized processes
- 1 shared listener, which must be omitted
- 4 listeners without a readable owner
- 2 Compose projects with 3 containers each
- 2 independent containers

`TestControlledWorkload` verifies the workload shape and its expected output (7 top-level services). `internal/tray`'s `layoutServices()` mirrors the same shape for the menu.

## Budgets

The budgets live in code, with margin for Go toolchain updates:

| Budget | allocs/op | B/op |
|---|---:|---:|
| Discovery (full pipeline) | ≤ 15,200 | ≤ 1,400,000 |
| Layout (menu update + render) | ≤ 245 | ≤ 15,500 |

Observed values and verification instructions live in [baseline.md](baseline.md).

Raising a budget is a product decision: it requires a change that grows real work and must be stated in the PR. Lowering one after an optimization is welcome.

For ad hoc inspection:

```sh
go test ./internal/monitor -run '^$' -bench 'BenchmarkControlledWorkload' -benchmem
go test ./internal/tray -run '^$' -bench '^BenchmarkLayout$' -benchmem
```

## Compare a change

The comparator answers one question: does the candidate do more work than the base? Its verdict is reproducible on any machine.

```sh
python3 benchmarks/compare.py /absolute/path/to/base /absolute/path/to/candidate
```

It requires both revisions to contain identical benchmark and fixture sources. Those sources are discovered by the `*perf_test.go` naming convention in `internal/monitor` and `internal/tray`, so a new fixture is protected automatically. The comparator compiles each revision's test and release binaries with the same Go toolchain, and reports `allocs/op`, `B/op` and binary size. It exits non-zero when the candidate allocates more per operation, allocates more bytes (1% tolerance), or produces a larger release binary. Base and candidate run in the same invocation, so hardware, OS and Go version are shared conditions, not baseline data.

Raw one-off notes may be kept under `benchmarks/experiments/` (ignored by Git). Anything that cannot be reproduced identically on another machine does not belong in the repository.
