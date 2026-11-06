#!/usr/bin/env python3
"""Sample one DevLinger process on a private D-Bus session; output JSON."""

import argparse
import json
import os
import signal
import subprocess
import time


def sample(pid):
    with open(f"/proc/{pid}/stat", encoding="utf-8") as file:
        fields = file.read().rsplit(")", 1)[1].split()
    with open(f"/proc/{pid}/status", encoding="utf-8") as file:
        status = dict(line.split(":", 1) for line in file if ":" in line)
    return (int(fields[11]) + int(fields[12]),
            int(status["VmRSS"].split()[0]), int(status["VmHWM"].split()[0]))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("watcher", help="path to compiled private watcher")
    parser.add_argument("binary", help="path to DevLinger binary")
    parser.add_argument("--seconds", type=float, default=35)
    args = parser.parse_args()
    if args.seconds <= 0:
        parser.error("--seconds must be positive")
    if not os.environ.get("DBUS_SESSION_BUS_ADDRESS"):
        parser.error("run under dbus-run-session")

    watcher = subprocess.Popen([args.watcher], stdout=subprocess.PIPE, text=True)
    app = None
    try:
        if watcher.stdout.readline().strip() != "ready":
            raise RuntimeError("watcher failed to start")
        started = time.monotonic()
        app = subprocess.Popen([args.binary])
        samples = []
        while time.monotonic() - started < args.seconds:
            if app.poll() is not None:
                raise RuntimeError(f"application exited early: {app.returncode}")
            try:
                samples.append((time.monotonic(), *sample(app.pid)))
            except (FileNotFoundError, KeyError) as error:
                if app.poll() is not None:
                    raise RuntimeError(f"application exited early: {app.returncode}") from error
                raise RuntimeError(f"failed to sample application process {app.pid}: {error}") from error
            time.sleep(0.2)
        if len(samples) < 2:
            raise RuntimeError("not enough process samples")
        first, last = samples[0], samples[-1]
        ticks = os.sysconf("SC_CLK_TCK")
        print(json.dumps({
            "elapsed_seconds": round(last[0] - first[0], 3),
            "cpu_seconds": round((last[1] - first[1]) / ticks, 3),
            "cpu_percent_one_core": round(100 * (last[1] - first[1]) / ticks / (last[0] - first[0]), 3),
            "rss_last_kib": last[2],
            "rss_sampled_max_kib": max(s[2] for s in samples),
            "rss_hwm_kib": max(s[3] for s in samples),
            "samples": len(samples),
        }))
    finally:
        for process in (app, watcher):
            if process and process.poll() is None:
                process.send_signal(signal.SIGTERM)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()


if __name__ == "__main__":
    main()
