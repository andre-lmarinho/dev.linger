# Baseline

These are the reference numbers of DevLinger's current performance baseline. Every developer who checks out this repository and repeats the commands below gets the same values — the indicators are hardware-independent by construction. Elapsed time is not a baseline number: it is machine-local and belongs to no contract. See [README.md](README.md).

Measured for the `v0.4.1` release candidate on 2026-11-18 with Go 1.27.3 and the fixed `BenchmarkControlledWorkload` fixture.

## Indicators

| Indicator | Observed | Budget (enforced by `go test ./...`) | Reproducibility |
|---|---:|---:|---|
| Discovery `allocs/op` | 14,881 | ≤ 15,200 | Exact on any machine with Go 1.27.3 |
| Discovery `B/op` | ~1,23–1,31 MB | ≤ 1,400,000 | Varies a few % with GC state; ceiling is the contract |
| Layout `allocs/op` | 235 | ≤ 245 | Exact on any machine with Go 1.27.3 |
| Layout `B/op` | 14,681 | ≤ 15,500 | Exact on any machine with Go 1.27.3 |
| Release binary | 8,495,367 bytes | comparator verdict | Exact for the same Go toolchain |

## How to verify

Any machine using Go 1.27.3, no fixtures or worktrees needed:

```sh
go test ./...
go test ./internal/monitor -v -run '^TestDiscoveryBudget$'
go test ./internal/tray -v -run '^TestLayoutBudget$'
```

The budget tests fail when the workload exceeds a budget, and print the observed values next to the budgets with `-v`. Any deviation beyond the budgets is a regression, wherever it appears.

To compare a change against its base revision, on one machine, in one invocation:

```sh
python3 benchmarks/compare.py /absolute/path/to/base /absolute/path/to/candidate
```

The comparator reports `allocs/op`, `B/op` and release binary bytes for both revisions and exits non-zero on regression. Timings are deliberately excluded.

## Updating this file

Raise a budget only for an explained product change, stated in the PR. Lowering one after an optimization is welcome — update the observed column and the budget together in the same commit.
