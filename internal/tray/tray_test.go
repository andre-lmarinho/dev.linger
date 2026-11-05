package tray

import (
	"os"
	"testing"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
)

type watcher struct{}

func (watcher) RegisterStatusNotifierItem(string) *dbus.Error { return nil }

func TestFormatCompose(t *testing.T) {
	s := monitor.Service{Name: "demo", Process: "Docker", Members: []monitor.Container{{ID: "a", Name: "web", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {ID: "b", Name: "db"}}}
	if got := formatService(s); got != "demo · Docker\nCPU 0.0% · Memory 0.0 MB" {
		t.Fatal(got)
	}
}

func TestFormatServiceCompact(t *testing.T) {
	s := monitor.Service{Name: "Vite", Project: "/home/user/shop", Process: "node", PID: 42, Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}, {Address: "::1", Port: 3000}}}
	if got := formatService(s); got != "shop · Vite · 3000\nCPU 0.0% · Memory 0.0 MB" {
		t.Fatal(got)
	}
}

func TestFormatMetrics(t *testing.T) {
	cpu, memory := 12.5, 23.0
	s := monitor.Service{Name: "Vite", CPU: &cpu, MemoryMB: &memory}
	if got := formatService(s); got != "Vite\nCPU 12.5% · Memory 23.0 MB" {
		t.Fatal(got)
	}
	s.CPU = nil
	if got := formatService(s); got != "Vite\nCPU 0.0% · Memory 23.0 MB" {
		t.Fatal(got)
	}
}

func TestServiceSubmenu(t *testing.T) {
	cpu, memory := 1.0, 2.0
	tray := &Tray{services: []monitor.Service{{Name: "Vite", CPU: &cpu, MemoryMB: &memory, Subservices: []monitor.Service{{Name: "HMR", CPU: &cpu, MemoryMB: &memory}}}}}
	items := tray.items()
	if items[1].Properties["label"].Value() != "Vite\nCPU 1.0% · Memory 2.0 MB\n1 sub-services" || items[1].Properties["enabled"].Value() != true || len(items[1].Children) != 1 {
		t.Fatalf("unexpected service submenu: %#v", items[1])
	}
	child := items[1].Children[0].Value().(layout)
	if child.Properties["label"].Value() != "HMR" {
		t.Fatal(child.Properties["label"].Value())
	}
}

func TestServiceItemsHaveStatusIcon(t *testing.T) {
	tray := &Tray{services: []monitor.Service{{Name: "Vite"}}}
	items := tray.items()
	if items[1].Properties["toggle-type"].Value() != "radio" || items[1].Properties["toggle-state"].Value() != int32(1) {
		t.Fatal("service status icon missing")
	}
}

func TestComposeMenu(t *testing.T) {
	cpu, memory := 12.0, 300.0
	tray := &Tray{services: []monitor.Service{{Name: "demo", Process: "Docker", CPU: &cpu, MemoryMB: &memory, Members: []monitor.Container{{Name: "web", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {Name: "db"}}}}}
	_, root, err := tray.GetLayout(0, -1, nil)
	if err != nil {
		t.Fatal(err)
	}
	project := root.Children[1].Value().(layout)
	if project.Properties["label"].Value() != "demo · Docker\nCPU 12.0% · Memory 300.0 MB\n2 containers" || project.Properties["enabled"].Value() != true || len(project.Children) != 2 {
		t.Fatalf("unexpected project: %#v", project)
	}
	web := project.Children[0].Value().(layout)
	if web.Properties["label"].Value() != "web · 3000" {
		t.Fatal(web)
	}
	_, shallow, err := tray.GetLayout(0, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(shallow.Children[1].Value().(layout).Children) != 0 {
		t.Fatal("depth ignored")
	}
	_, children, err := tray.GetLayout(project.ID, 1, nil)
	if err != nil || len(children.Children) != 2 {
		t.Fatalf("submenu: %#v %v", children, err)
	}
	value, err := tray.GetProperty(web.ID, "label")
	if err != nil || value.Value() != "web · 3000" {
		t.Fatalf("property: %v %v", value, err)
	}
}

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
	icon := func() []pixmap {
		t.Helper()
		var value dbus.Variant
		if err := conn.Object(tray.conn.Names()[0], itemPath).Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierItem", "IconPixmap").Store(&value); err != nil {
			t.Fatal(err)
		}
		var images []pixmap
		if err := dbus.Store([]interface{}{value.Value()}, &images); err != nil {
			t.Fatal(err)
		}
		return images
	}
	if got := icon()[0].Data; got[(7*16+7)*4] != 160 {
		t.Fatal("initial icon is not dimmed")
	}
	tray.Update([]monitor.Service{{Name: "node", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {Name: "demo", Process: "Docker", Members: []monitor.Container{{Name: "web"}}}}, nil)
	if got := icon()[0].Data; got[(7*16+7)*4] != 255 {
		t.Fatal("active icon is not bright")
	}
	var revision uint32
	var menu layout
	err = conn.Object(tray.conn.Names()[0], menuPath).Call("com.canonical.dbusmenu.GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &menu)
	if err != nil {
		t.Fatal(err)
	}
	if revision != 1 || len(menu.Children) != 4 {
		t.Fatalf("unexpected menu: %#v revision %d", menu, revision)
	}
	err = conn.Object(tray.conn.Names()[0], menuPath).Call("com.canonical.dbusmenu.GetLayout", 0, int32(11), int32(1), []string{}).Store(&revision, &menu)
	if err != nil || len(menu.Children) != 1 {
		t.Fatalf("submenu: %#v %v", menu, err)
	}
	tray.Update(nil, nil)
	if got := icon()[0].Data; got[(7*16+7)*4] != 160 {
		t.Fatal("empty icon is not dimmed")
	}
}
