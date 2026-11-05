package monitor

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type cpuSample struct {
	start, ticks uint64
}

// Metrics retains only the previous CPU counters needed for the next sample.
type Metrics struct {
	total     uint64
	idle      uint64
	processes map[int]cpuSample
}

func (m *Metrics) Update(services []Service) {
	m.update("/proc", services)
}

func (m *Metrics) update(proc string, services []Service) {
	data, _ := os.ReadFile(filepath.Join(proc, "stat"))
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	var total, idle uint64
	for i, field := range fields[min(1, len(fields)):min(len(fields), 9)] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			total, idle = 0, 0
			break
		}
		total += value
		if i == 3 || i == 4 {
			idle += value
		}
	}
	cores := runtime.NumCPU()
	hostCPU, hostMemory := 0.0, 0.0
	var systemUptime float64
	if data, err := os.ReadFile(filepath.Join(proc, "uptime")); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			systemUptime, _ = strconv.ParseFloat(fields[0], 64)
		}
	}
	if m.total != 0 && total > m.total && idle >= m.idle && idle-m.idle <= total-m.total {
		hostCPU = 100 * float64((total-m.total)-(idle-m.idle)) / float64(total-m.total)
	}
	if meminfo, err := os.ReadFile(filepath.Join(proc, "meminfo")); err == nil {
		var memoryTotal, available uint64
		for _, line := range strings.Split(string(meminfo), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 3 || fields[2] != "kB" {
				continue
			}
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			switch fields[0] {
			case "MemTotal:":
				memoryTotal = value
			case "MemAvailable:":
				available = value
			}
		}
		if memoryTotal >= available {
			hostMemory = float64(memoryTotal-available) * 1024 / 1_000_000
		}
	}
	next := make(map[int]cpuSample)
	for i := range services {
		s := &services[i]
		if s.Docker {
			s.CPU, s.MemoryMB = hostCPU, hostMemory
			continue
		}
		s.CPU, s.MemoryMB = 0, 0
		// shortcut: Linux /proc start times use USER_HZ (100), revisit for non-Linux ports.
		if started := float64(s.start) / 100; systemUptime >= started {
			s.Uptime = time.Duration(systemUptime-started) * time.Second
		}
		if s.PID <= 0 {
			continue
		}
		base := filepath.Join(proc, strconv.Itoa(s.PID))
		stat, err := os.ReadFile(filepath.Join(base, "stat"))
		if err != nil {
			continue
		}
		end := strings.LastIndexByte(string(stat), ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(stat[end+1:]))
		if len(fields) < 20 {
			continue
		}
		user, e1 := strconv.ParseUint(fields[11], 10, 64)
		system, e2 := strconv.ParseUint(fields[12], 10, 64)
		start, e3 := strconv.ParseUint(fields[19], 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || start != s.start {
			continue
		}
		current := cpuSample{start: start, ticks: user + system}
		next[s.PID] = current
		if previous, ok := m.processes[s.PID]; ok && previous.start == start && current.ticks >= previous.ticks && total > m.total && m.total != 0 && cores > 0 {
			s.CPU = float64(current.ticks-previous.ticks) / float64(total-m.total) * float64(cores) * 100
		}
		statm, err := os.ReadFile(filepath.Join(base, "statm"))
		fields = strings.Fields(string(statm))
		if err == nil && len(fields) > 1 {
			if pages, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				s.MemoryMB = float64(pages*uint64(os.Getpagesize())) / 1_000_000
			}
		}
	}
	m.total, m.idle, m.processes = total, idle, next
}
