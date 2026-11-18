package tray

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
)

type watcher struct {
	registered chan string
}

func (w watcher) RegisterStatusNotifierItem(name string) *dbus.Error {
	if w.registered != nil {
		select {
		case w.registered <- name:
		default:
		}
	}
	return nil
}

func TestFormatServiceCompact(t *testing.T) {
	s := monitor.Service{Name: "Vite", Project: "/home/user/shop", PID: 42, Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}, {Address: "::1", Port: 3000}}}
	if got := formatServiceName(s); got != "shop · :3000" {
		t.Fatal(got)
	}
}

func TestStandaloneStopActionIsTopLevel(t *testing.T) {
	tray := &Tray{}
	tray.setServices([]monitor.Service{{Name: "Vite", PID: 42}})
	items := tray.items()
	if len(items) != 2 || len(items[1].Children) != 0 {
		t.Fatalf("unexpected standalone menu: %#v", items)
	}
	if items[1].ID != tray.actionIDs[0] || items[1].Properties["label"].Value() != "Vite" || items[1].Properties["icon-name"].Value() != "process-stop" {
		t.Fatalf("service row is not the stop action: %#v", items[1])
	}
}

func TestSubServicesAreNotShown(t *testing.T) {
	tray := &Tray{}
	tray.setServices([]monitor.Service{{Name: "Vite", Members: []monitor.Service{{Name: "HMR"}}}})
	items := tray.items()
	if items[1].Properties["label"].Value() != "Vite" || len(items[1].Children) != 0 {
		t.Fatalf("unexpected service row: %#v", items[1])
	}
}

func TestComposeMenu(t *testing.T) {
	tray := &Tray{}
	tray.setServices([]monitor.Service{{Name: "demo", Docker: true, Members: []monitor.Service{{Name: "web", Endpoints: []monitor.Endpoint{{Address: "127.0.0.1", Port: 3000}}}, {Name: "db"}}}})
	_, root, err := tray.GetLayout(0, -1, nil)
	if err != nil {
		t.Fatal(err)
	}
	project := root.Children[1].Value().(layout)
	if project.Properties["label"].Value() != "demo · Docker" || project.Properties["enabled"].Value() != true || len(project.Children) != 0 {
		t.Fatalf("unexpected project: %#v", project)
	}
	_, shallow, err := tray.GetLayout(0, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(shallow.Children[1].Value().(layout).Children) != 0 {
		t.Fatal("depth ignored")
	}
	_, children, err := tray.GetLayout(project.ID, 1, nil)
	if err != nil || len(children.Children) != 0 {
		t.Fatalf("submenu: %#v %v", children, err)
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
	tray, err := New(context.Background())
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
	if revision != 1 || len(menu.Children) != 4 {
		t.Fatalf("unexpected menu: %#v revision %d", menu, revision)
	}
	err = conn.Object(tray.conn.Names()[0], menuPath).Call("com.canonical.dbusmenu.GetLayout", 0, tray.actionIDs[1], int32(1), []string{}).Store(&revision, &menu)
	if err != nil || len(menu.Children) != 0 {
		t.Fatalf("service row: %#v %v", menu, err)
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
			{Name: "Vite"},
			{Name: "demo", Docker: true, Members: []monitor.Service{{Name: "web"}}},
		},
	}
	for name, services := range cases {
		tray := &Tray{}
		tray.setServices(services)
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

func TestClickStopsMappedService(t *testing.T) {
	called := make(chan monitor.Service, 1)
	tray := &Tray{
		ctx: context.Background(),
		stop: func(_ context.Context, service monitor.Service) error {
			called <- service
			return nil
		},
	}
	tray.setServices([]monitor.Service{{Name: "Vite", PID: 42}})
	actionID := tray.actionIDs[0]

	if err := tray.Event(actionID, "clicked", dbus.Variant{}, 0); err != nil {
		t.Fatal(err)
	}
	select {
	case service := <-called:
		if service.PID != 42 {
			t.Fatalf("stopped wrong service: %#v", service)
		}
	case <-time.After(time.Second):
		t.Fatal("stop action was not called")
	}
}

func TestStaleActionDoesNotStopAnotherService(t *testing.T) {
	called := make(chan monitor.Service, 1)
	tray := &Tray{
		stop: func(_ context.Context, service monitor.Service) error {
			called <- service
			return nil
		},
	}
	tray.setServices([]monitor.Service{{Name: "old", PID: 1}})
	staleID := tray.actionIDs[0]
	tray.setServices([]monitor.Service{{Name: "new", PID: 2}})

	tray.Event(staleID, "clicked", dbus.Variant{}, 0)
	select {
	case service := <-called:
		t.Fatalf("stale action stopped %#v", service)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNewWaitsForWatcher(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	type result struct {
		tray *Tray
		err  error
	}

	done := make(chan result, 1)

	go func() {
		tray, err := New(ctx)
		done <- result{tray, err}
	}()

	select {
	case r := <-done:
		if r.tray != nil {
			r.tray.Close()
		}
		t.Fatalf("New returned before watcher became available: %v", r.err)
	case <-time.After(100 * time.Millisecond):
		// Expected: New is still waiting.
	}

	reply, err := conn.RequestName(
		"org.kde.StatusNotifierWatcher",
		dbus.NameFlagDoNotQueue,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		t.Skip("watcher already running; use isolated D-Bus session")
	}

	if err := conn.Export(
		watcher{},
		"/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher",
	); err != nil {
		t.Fatal(err)
	}

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		defer r.tray.Close()
	case <-ctx.Done():
		t.Fatal("New did not register after watcher became available")
	}
}

func TestNewStopsWaitingWhenContextCanceled(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		tray, err := New(ctx)
		if tray != nil {
			tray.Close()
		}
		done <- err
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("New did not stop after context cancellation")
	}
}

func TestTrayRegistersAfterWatcherRestart(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}

	const watcherName = "org.kde.StatusNotifierWatcher"
	const watcherPath = "/StatusNotifierWatcher"

	first, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	second, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	registeredFirst := make(chan string, 1)
	registeredSecond := make(chan string, 1)

	startWatcher := func(conn *dbus.Conn, registered chan string) {
		t.Helper()

		if err := conn.Export(
			watcher{registered: registered},
			watcherPath,
			watcherName,
		); err != nil {
			t.Fatal(err)
		}

		reply, err := conn.RequestName(
			watcherName,
			dbus.NameFlagDoNotQueue,
		)
		if err != nil {
			t.Fatal(err)
		}
		if reply != dbus.RequestNameReplyPrimaryOwner {
			t.Fatal("watcher name already owned")
		}
	}

	startWatcher(first, registeredFirst)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tray, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tray.Close()

	select {
	case <-registeredFirst:
	case <-time.After(time.Second):
		t.Fatal("initial watcher did not receive registration")
	}

	if _, err := first.ReleaseName(watcherName); err != nil {
		t.Fatal(err)
	}

	startWatcher(second, registeredSecond)

	select {
	case <-registeredSecond:
		// Expected: DevLinger registered again.
	case <-time.After(2 * time.Second):
		t.Fatal("DevLinger did not register after watcher restart")
	}
}

func TestIsWatcherUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "service unknown",
			err:  dbus.NewError("org.freedesktop.DBus.Error.ServiceUnknown", nil),
			want: true,
		},
		{
			name: "name has no owner",
			err:  dbus.NewError("org.freedesktop.DBus.Error.NameHasNoOwner", nil),
			want: true,
		},
		{
			name: "unknown method",
			err:  dbus.NewError("org.freedesktop.DBus.Error.UnknownMethod", nil),
		},
		{
			name: "unknown interface",
			err:  dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", nil),
		},
		{
			name: "invalid arguments",
			err:  dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", nil),
		},
		{
			name: "no such object",
			err:  dbus.NewError("org.freedesktop.DBus.Error.NoSuchObject", nil),
		},
		{
			name: "generic error",
			err:  errors.New("connection closed"),
		},
		{
			name: "service unknown as value",
			err: dbus.Error{
				Name: "org.freedesktop.DBus.Error.ServiceUnknown",
			},
			want: true,
		},
		{
			name: "name has no owner as value",
			err: dbus.Error{
				Name: "org.freedesktop.DBus.Error.NameHasNoOwner",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWatcherUnavailable(tt.err)
			if got != tt.want {
				t.Errorf("isWatcherUnavailable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type rejectingWatcher struct{}

func (rejectingWatcher) RegisterStatusNotifierItem(string) *dbus.Error {
	return dbus.NewError(
		"org.freedesktop.DBus.Error.InvalidArgs",
		[]interface{}{"invalid indicator"},
	)
}

func TestNewFailsOnPermanentWatcherError(t *testing.T) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.Export(
		rejectingWatcher{},
		"/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher",
	); err != nil {
		t.Fatal(err)
	}

	reply, err := conn.RequestName(
		"org.kde.StatusNotifierWatcher",
		dbus.NameFlagDoNotQueue,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatal("watcher name already owned")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = New(ctx)

	if err == nil {
		t.Fatal("expected permanent registration error")
	}

	var dbusErr dbus.Error
	if !errors.As(err, &dbusErr) {
		t.Fatalf("expected D-Bus error, got %v", err)
	}

	if dbusErr.Name != "org.freedesktop.DBus.Error.InvalidArgs" {
		t.Fatalf("unexpected D-Bus error: %s", dbusErr.Name)
	}
}

func TestIsWatcherAcquired(t *testing.T) {
	tests := []struct {
		name   string
		signal *dbus.Signal
		want   bool
	}{
		{
			name: "nil signal",
		},
		{
			name: "wrong signal",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameLost",
				Body: []interface{}{
					"org.kde.StatusNotifierWatcher", "", ":1.42",
				},
			},
		},
		{
			name: "invalid body",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameOwnerChanged",
				Body: []interface{}{"org.kde.StatusNotifierWatcher"},
			},
		},
		{
			name: "different service",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameOwnerChanged",
				Body: []interface{}{"org.example.Other", "", ":1.42"},
			},
		},
		{
			name: "watcher lost",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameOwnerChanged",
				Body: []interface{}{"org.kde.StatusNotifierWatcher", ":1.41", ""},
			},
		},
		{
			name: "watcher acquired",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameOwnerChanged",
				Body: []interface{}{"org.kde.StatusNotifierWatcher", "", ":1.42"},
			},
			want: true,
		},
		{
			name: "watcher replaced",
			signal: &dbus.Signal{
				Name: "org.freedesktop.DBus.NameOwnerChanged",
				Body: []interface{}{"org.kde.StatusNotifierWatcher", ":1.41", ":1.42"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWatcherAcquired(tt.signal); got != tt.want {
				t.Errorf("isWatcherAcquired() = %v, want %v", got, tt.want)
			}
		})
	}
}
