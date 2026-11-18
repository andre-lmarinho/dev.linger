package tray

import (
	"testing"

	"github.com/devlinger/devlinger/internal/monitor"
)

// layoutServices mirrors the controlled monitor workload: three local
// servers, two Compose groups and two standalone containers.
func layoutServices() []monitor.Service {
	return []monitor.Service{
		{Name: "Vite", Project: "/work/alpha", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}},
		{Name: "Django", Project: "/work/beta", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 8000}}},
		{Name: "Storybook", Project: "/work/gamma", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 6006}}},
		{Name: "compose-alpha", Docker: true, Members: []monitor.Service{{ContainerID: "alpha-1"}, {ContainerID: "alpha-2"}, {ContainerID: "alpha-3"}}},
		{Name: "compose-beta", Docker: true, Members: []monitor.Service{{ContainerID: "beta-1"}, {ContainerID: "beta-2"}, {ContainerID: "beta-3"}}},
		{Name: "standalone-a", Docker: true, ContainerID: "standalone-a"},
		{Name: "standalone-b", Docker: true, ContainerID: "standalone-b"},
	}
}

// BenchmarkLayout measures the full menu update: service bookkeeping plus
// the complete D-BusMenu layout render.
func BenchmarkLayout(b *testing.B) {
	services := layoutServices()
	t := &Tray{}
	b.ReportAllocs()
	for b.Loop() {
		t.setServices(services)
		t.root()
	}
}
