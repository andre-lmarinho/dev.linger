package tray

import (
	"os"
	"testing"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
)

type watcher struct{}

func (watcher) RegisterStatusNotifierItem(string) *dbus.Error { return nil }

func TestMenuOverDBus(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reply, err := conn.RequestName("org.kde.StatusNotifierWatcher", dbus.NameFlagDoNotQueue)
	if err != nil {
		t.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		t.Skip("a watcher is already running")
	}
	if err := conn.Export(watcher{}, "/StatusNotifierWatcher", "org.kde.StatusNotifierWatcher"); err != nil {
		t.Fatal(err)
	}
	tray, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer tray.Close()
	tray.Update([]monitor.Service{{Name: "node", Port: 3000, Address: "127.0.0.1"}}, nil)
	var revision uint32
	var menu layout
	err = conn.Object(tray.conn.Names()[0], menuPath).Call("com.canonical.dbusmenu.GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &menu)
	if err != nil {
		t.Fatal(err)
	}
	if revision != 1 || len(menu.Children) != 4 {
		t.Fatalf("unexpected menu: %#v revision %d", menu, revision)
	}
}
