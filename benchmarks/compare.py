#!/usr/bin/env python3

"""Paired benchmark comparison with hardware-independent indicators only.

Reports allocations, bytes per operation and release binary size for a base
and a candidate revision, and exits non-zero when the candidate does more
work. Elapsed time is deliberately excluded: it is machine-local and not a
DevLinger indicator.
"""

import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

SOURCES = (
    "internal/monitor/fixture_perf_test.go",
    "internal/monitor/perf_test.go",
    "internal/tray/perf_test.go",
)
PACKAGES = {
    "monitor": r"Benchmark(ControlledWorkload|Catalog)",
    "tray": r"^BenchmarkLayout$",
}
BENCHMARK = re.compile(
    r"^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+"
    r"(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op$"
)
BYTES_TOLERANCE = 0.01


def run(command, cwd, env=None):
    return subprocess.run(
        command, cwd=cwd, check=True, text=True, capture_output=True, env=env
    )


def verify_same_fixture(before, after):
    different = [
        path for path in SOURCES
        if (before / path).read_bytes() != (after / path).read_bytes()
    ]
    if different:
        raise SystemExit(
            "benchmark comparison requires identical fixtures:\n- "
            + "\n- ".join(different)
        )


def parse(output):
    result = {}
    for line in output.splitlines():
        match = BENCHMARK.match(line)
        if match:
            result[match.group(1)] = (
                float(match.group(2)),
                int(match.group(3)),
                int(match.group(4)),
            )
    return result


def delta(before, after):
    if before == 0:
        return "—"
    return f"{(after - before) / before:+.1%}"


def main():
    if len(sys.argv) != 3:
        raise SystemExit(f"usage: {sys.argv[0]} BASE_WORKTREE CANDIDATE_WORKTREE")

    roots = {
        "base": Path(sys.argv[1]).resolve(),
        "candidate": Path(sys.argv[2]).resolve(),
    }
    verify_same_fixture(roots["base"], roots["candidate"])

    # Pin one toolchain for both revisions: independent per-directory
    # selection could otherwise compare binaries built by different Go
    # versions.
    selected = run(["go", "env", "GOVERSION"], Path.cwd()).stdout.strip()
    env = {**os.environ, "GOTOOLCHAIN": selected}
    print(f"toolchain: {selected}")

    def build(command, version, root):
        try:
            run(command, root, env)
        except subprocess.CalledProcessError as error:
            raise SystemExit(
                f"{version}: cannot build with the selected toolchain {selected}:\n"
                f"{(error.stderr or '').strip()}"
            ) from error

    with tempfile.TemporaryDirectory(prefix="devlinger-benchmark-") as tmp:
        tmp = Path(tmp)
        measurements = {}
        sizes = {}
        for version, root in roots.items():
            combined = {}
            for package, pattern in PACKAGES.items():
                binary = tmp / f"{version}-{package}.test"
                build(
                    ["go", "test", "-c", "-o", str(binary), f"./internal/{package}"],
                    version,
                    root,
                )
                output = run([
                    str(binary),
                    "-test.run", "^$",
                    "-test.bench", pattern,
                    "-test.benchmem",
                    "-test.benchtime", "1s",
                ], root).stdout
                parsed = parse(output)
                if not parsed:
                    raise SystemExit(f"{version}: no benchmarks matched {pattern!r}")
                combined.update(parsed)

            release = tmp / f"{version}-release"
            build([
                "go", "build", "-trimpath", "-ldflags=-s -w",
                "-o", str(release), ".",
            ], version, root)
            sizes[version] = release.stat().st_size
            measurements[version] = combined

    if measurements["base"].keys() != measurements["candidate"].keys():
        raise SystemExit("base and candidate produced different benchmark names")

    regressions = []
    print("| Benchmark | Metric | Base | Candidate | Change | Verdict |")
    print("|---|---:|---:|---:|---:|---|")
    for name in sorted(measurements["base"]):
        short = name.removeprefix("Benchmark")
        for column, label in enumerate(("ns/op", "B/op", "allocs/op")):
            if label == "ns/op":
                continue  # machine-local; not an indicator
            before = measurements["base"][name][column]
            after = measurements["candidate"][name][column]
            verdict = "OK"
            if label == "allocs/op" and after > before:
                verdict = "REGRESSION"
            if label == "B/op" and after > before * (1 + BYTES_TOLERANCE):
                verdict = "REGRESSION"
            if verdict == "REGRESSION":
                regressions.append(f"{short} {label}: {before} -> {after}")
            print(
                f"| {short} | {label} | {before:g} | {after:g} | "
                f"{delta(before, after)} | {verdict} |"
            )

    before, after = sizes["base"], sizes["candidate"]
    verdict = "REGRESSION" if after > before else "OK"
    if verdict == "REGRESSION":
        regressions.append(f"Release binary: {before} -> {after}")
    print(f"| Release binary | bytes | {before} | {after} | {delta(before, after)} | {verdict} |")

    if regressions:
        print("\nRegressions:", "; ".join(regressions), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
