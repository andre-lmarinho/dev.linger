package tray

import (
	"os"
	"testing"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
)

// Run under dbus-run-session to avoid interfering with the desktop watcher.
func BenchmarkUpdate(b *testing.B) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		b.Skip("requires a private session bus")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()
	reply, err := conn.RequestName("org.kde.StatusNotifierWatcher", dbus.NameFlagDoNotQueue)
	if err != nil {
		b.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		b.Skip("requires a private session bus without a watcher")
	}
	if err := conn.Export(watcher{}, "/StatusNotifierWatcher", "org.kde.StatusNotifierWatcher"); err != nil {
		b.Fatal(err)
	}
	t, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer t.Close()
	services := []monitor.Service{{Name: "Vite", Project: "/project", CPUValid: true, MemoryValid: true}}
	b.ReportAllocs()
	for b.Loop() {
		t.Update(services, nil)
	}
}
