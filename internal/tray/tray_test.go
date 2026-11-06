package tray

import (
	"os"
	"testing"
	"time"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
)

type watcher struct{}

func (watcher) RegisterStatusNotifierItem(string) *dbus.Error { return nil }

func TestFormatServiceCompact(t *testing.T) {
	s := monitor.Service{Name: "Vite", Project: "/home/user/shop", PID: 42, Branch: "main", Uptime: 20 * time.Minute, Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}, {Address: "::1", Port: 3000}}}
	if got := formatService(s); got != "shop · Vite · :3000 · main · 20m" {
		t.Fatal(got)
	}
}

func TestFormatAvailableMetrics(t *testing.T) {
	s := monitor.Service{Name: "Vite", CPUValid: true, MemoryValid: true}
	if got := formatService(s); got != "Vite\nCPU 0.0% · Memory 0 MB" {
		t.Fatal(got)
	}
	s.CPUValid = false
	if got := formatService(s); got != "Vite\nMemory 0 MB" {
		t.Fatal(got)
	}
}

func TestServiceSubmenu(t *testing.T) {
	tray := &Tray{services: []monitor.Service{{Name: "Vite", CPU: 1, MemoryMB: 2, CPUValid: true, MemoryValid: true, Members: []monitor.Service{{Name: "HMR", CPU: 1, MemoryMB: 2}}}}}
	items := tray.items()
	if items[1].Properties["label"].Value() != "Vite\nCPU 1.0% · Memory 2 MB\n1 sub-service" || items[1].Properties["enabled"].Value() != true || len(items[1].Children) != 1 {
		t.Fatalf("unexpected service submenu: %#v", items[1])
	}
	child := items[1].Children[0].Value().(layout)
	if child.Properties["label"].Value() != "↳ HMR" {
		t.Fatal(child.Properties["label"].Value())
	}
}

func TestComposeMenu(t *testing.T) {
	tray := &Tray{services: []monitor.Service{{Name: "demo", Docker: true, Members: []monitor.Service{{Name: "web", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {Name: "db"}}}}}
	_, root, err := tray.GetLayout(0, -1, nil)
	if err != nil {
		t.Fatal(err)
	}
	project := root.Children[1].Value().(layout)
	if project.Properties["label"].Value() != "demo · Docker\n2 containers" || project.Properties["enabled"].Value() != true || project.Properties["toggle-type"].Value() != "radio" || len(project.Children) != 2 {
		t.Fatalf("unexpected project: %#v", project)
	}
	web := project.Children[0].Value().(layout)
	if web.Properties["label"].Value() != "↳ web · :3000" {
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
	if err != nil || value.Value() != "↳ web · :3000" {
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
	tray.Update([]monitor.Service{{Name: "node", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {Name: "demo", Docker: true, Members: []monitor.Service{{Name: "web"}}}}, nil)
	if got := icon()[0].Data; got[(7*16+7)*4] != 255 {
		t.Fatal("active icon is not bright")
	}
	var revision uint32
	var menu layout
	err = conn.Object(tray.conn.Names()[0], menuPath).Call("com.canonical.dbusmenu.GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &menu)
	if err != nil {
		t.Fatal(err)
	}
	if revision != 1 || len(menu.Children) != 3 {
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

// TestMenuHasNoQuitNoAutostart locks in the requirement: the menu must never
// contain a Quit item or an autostart toggle, in any state.
func TestMenuHasNoQuitNoAutostart(t *testing.T) {
	cases := map[string][]monitor.Service{
		"empty": nil,
		"services": {
			{Name: "Vite", CPUValid: true},
			{Name: "demo", Docker: true, Members: []monitor.Service{{Name: "web"}}},
		},
	}
	for name, services := range cases {
		tray := &Tray{services: services}
		var walk func(children []dbus.Variant)
		walk = func(children []dbus.Variant) {
			for _, v := range children {
				item := v.Value().(layout)
				label, _ := item.Properties["label"].Value().(string)
				if label == "Quit" || label == "Start at login" {
					t.Fatalf("%s: forbidden item %q in menu", name, label)
				}
				walk(item.Children)
			}
		}
		walk(tray.root().Children)
	}
}
